/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"github.com/hopeio/initialize/conf_center"
	daopkg "github.com/hopeio/initialize/dao"
	"github.com/hopeio/initialize/rootconf"
	"github.com/hopeio/utils/errors/multierr"
	"github.com/hopeio/utils/log"
	"github.com/hopeio/utils/os/fs"
	pathi "github.com/hopeio/utils/os/fs/path"
	"github.com/spf13/viper"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
)

// globalConfig
// 全局配置
type globalConfig[C Config, D Dao] struct {
	RootConfig    rootconf.RootConfig `mapstructure:",squash"`
	BuiltinConfig builtinConfig

	Config C
	Dao    D

	*viper.Viper
	/*
		cacheConf      any*/
	editTimes   uint32
	defers      []func()
	initialized bool
	mu          sync.RWMutex

	// 为后续仍有需要注入的config和dao保留的后门,与Inject(Config,Dao) 配合
	injectConfs []Config
	injectDaos  []Dao
}

func newGlobal[C Config, D Dao]() *globalConfig[C, D] {
	gc := &globalConfig[C, D]{
		RootConfig: rootconf.RootConfig{
			EnvConfig: rootconf.EnvConfig{Debug: true},
		},
		Viper: viper.NewWithOptions(viper.WithCodecRegistry(codecRegistry)),
	}
	return gc
}
func NewGlobalWith[C Config, D Dao](conf C, dao D, configCenter ...conf_center.ConfigCenter) *globalConfig[C, D] {
	gc := newGlobal[C, D]()
	gc.Config = conf
	gc.Dao = dao
	gc.init(configCenter...)
	return gc
}

// var Global = initialize.NewGlobal[C,D]()
func NewGlobal[C Config, D Dao](configCenter ...conf_center.ConfigCenter) *globalConfig[C, D] {
	gc := newGlobal[C, D]()
	v := reflect.ValueOf(&gc.Config).Elem()
	if v.Kind() == reflect.Struct {
		log.Fatalf("generic type should be a pointer type")
	}
	v.Set(reflect.New(reflect.TypeOf(gc.Config).Elem()))
	v = reflect.ValueOf(&gc.Dao).Elem()
	if v.Kind() == reflect.Struct {
		log.Fatalf("generic type should be a pointer type")
	}
	v.Set(reflect.New(reflect.TypeOf(gc.Dao).Elem()))
	gc.init(configCenter...)
	return gc
}

func Start[C Config, D Dao](conf C, dao D, configCenter ...conf_center.ConfigCenter) func() {
	gc := NewGlobalWith[C, D](conf, dao, configCenter...)
	return gc.Cleanup
}

func NewGlobal2[C Config](configCenter ...conf_center.ConfigCenter) *globalConfig[C, *EmbeddedPresets] {
	return NewGlobal[C, *EmbeddedPresets](configCenter...)
}

func (gc *globalConfig[C, D]) init(configCenter ...conf_center.ConfigCenter) {
	applyFlagConfig(gc.Viper, &gc.RootConfig)
	gc.RootConfig.AfterInject()
	// 为支持自定义配置中心,并且遵循依赖最小化原则,配置中心改为可插拔的,考虑将配置序列话也照此重做
	// 注册配置中心,默认注册本地文件
	for _, cc := range configCenter {
		conf_center.RegisterConfigCenter(cc)
	}

	gc.defers = append(gc.defers, func() {
		if err := closeDao(gc.Dao); err != nil {
			log.Errorf("close Dao error: %v", err)
		}
	})
	gc.loadConfig()
	gc.initialized = true
}

// var Global = initialize.NewGlobal[C,D]()
// func main(){
// 		defer global.Global.Cleanup()
// }

func (gc *globalConfig[C, D]) Cleanup() {
	if !gc.initialized {
		log.Fatalf("not initialize, please call initialize.initHandler or initialize.Start")
	}
	// 倒序调用defer
	for i := len(gc.defers) - 1; i > 0; i-- {
		gc.defers[i]()
	}
	if gc.RootConfig.ConfigCenter.ConfigCenter != nil {
		if err := gc.RootConfig.ConfigCenter.ConfigCenter.Close(); err != nil {
			log.Errorf("close config center error: %v", err)
		}
	}
	log.Sync()
}

const defaultConfigName = "config"

func (gc *globalConfig[C, D]) loadConfig() {
	executable, err := os.Executable()
	if err != nil {
		log.Fatalf("get executable error: %v", err)
	}
	gc.RootConfig.Executable = executable
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("get work dir error: %v", err)
	}
	gc.RootConfig.ExecDir = wd
	gc.Viper.AutomaticEnv()
	var format string
	// find config
	if gc.RootConfig.ConfPath == "" {
		log.Debugf("lack of flag -c or --config, searching 'config.*' in %s", wd)
		for _, ext := range viper.SupportedExts {
			filePath := filepath.Join(".", defaultConfigName+"."+ext)
			if fs.Exist(filePath) {
				log.Debugf("found file: '%s'", filePath)
				gc.RootConfig.ConfPath = filePath
				format = ext
				break
			}
		}
		// find in config dir
		if gc.RootConfig.ConfPath == "" {
			configDir := filepath.Join(wd, "config")
			fileInfo, err := os.Stat(configDir)
			if err == nil && fileInfo.IsDir() {
				for _, ext := range viper.SupportedExts {
					filePath := filepath.Join(configDir, defaultConfigName+"."+ext)
					if fs.Exist(filePath) {
						log.Debugf("found file: '%s'", filePath)
						gc.RootConfig.ConfPath = filePath
						format = ext
						break
					}
				}
			}
		}

		/*		if format == "" {
				log.Warn("not found config, use env and flag")
			}*/
	}
	if gc.RootConfig.ConfPath != "" {
		gc.RootConfig.ConfPath, err = filepath.Abs(gc.RootConfig.ConfPath)
		if err != nil {
			log.Fatalf("get abs path error: %v", err)
		}
		log.Infof("load config from: '%s'", gc.RootConfig.ConfPath)
		if format == "" {
			format = filepath.Ext(gc.RootConfig.ConfPath)
			if format != "" {
				// remove .
				format = format[1:]
				if !slices.Contains(viper.SupportedExts, format) {
					log.Fatalf("unsupport config format, support: %v", viper.SupportedExts)
				}
			} else {
				log.Fatalf("config path need format ext, support: %v", viper.SupportedExts)
			}
		}

		gc.RootConfig.ConfigCenter.Format = format
		gc.Viper.SetConfigType(format)
		gc.Viper.SetConfigFile(gc.RootConfig.ConfPath)
		err := gc.Viper.ReadInConfig()
		if err != nil {
			log.Fatal(err)
		}
	}

	gc.setRootConfig()
	gc.setEnvConfig()
	for i := range gc.RootConfig.NoInject {
		gc.RootConfig.NoInject[i] = strings.ToUpper(gc.RootConfig.NoInject[i])
	}

	var singleTemplateFileConfig bool
	if gc.RootConfig.EnvConfig.ConfigCenter.ConfigCenter == nil {
		if gc.RootConfig.Env == "" {
			singleTemplateFileConfig = true
		}
		if gc.RootConfig.ConfigCenter.Type != "" {
			gc.RootConfig.ConfigCenter.ConfigCenter = conf_center.GetConfigCenter(gc.RootConfig.ConfigCenter.Type)
		}
	}
	cfgcenter := gc.RootConfig.ConfigCenter.ConfigCenter
	if cfgcenter != nil {
		applyFlagConfig(gc.Viper, cfgcenter)
	}
	// hook function
	gc.beforeInjectCall(gc.Config, gc.Dao)
	gc.genConfigTemplate(singleTemplateFileConfig)
	localConfig := newLocal(gc.RootConfig.LocalConfig.ReloadInterval, gc.RootConfig.LocalConfig.Paths...)
	if gc.RootConfig.Env != "" {
		var defaultEnvConfigPath string
		if gc.RootConfig.ConfPath != "" {
			defaultEnvConfigPath = pathi.FileNoExt(gc.RootConfig.ConfPath) + "." + gc.RootConfig.Env + "." + gc.RootConfig.ConfigCenter.Format
		} else if gc.RootConfig.ConfigCenter.Format != "" {
			gc.Viper.SetConfigType(format)
			defaultEnvConfigPath = defaultConfigName + "." + gc.RootConfig.Env + "." + gc.RootConfig.ConfigCenter.Format
		} else {
			for _, ext := range viper.SupportedExts {
				filePath := filepath.Join(".", defaultConfigName+"."+gc.RootConfig.Env+"."+ext)
				if fs.Exist(filePath) {
					defaultEnvConfigPath = filePath
					log.Debugf("found file: '%s'", filePath)
					gc.RootConfig.ConfigCenter.Format = ext
					gc.Viper.SetConfigType(ext)
					break
				}
			}
		}

		if defaultEnvConfigPath != "" && fs.Exist(defaultEnvConfigPath) && !slices.Contains(localConfig.Paths, defaultEnvConfigPath) {
			localConfig.Paths = append([]string{defaultEnvConfigPath}, localConfig.Paths...)
		}
	}
	if len(localConfig.Paths) > 0 {
		err = localConfig.Handle(func(data io.Reader) error {
			gc.mu.TryLock()
			defer gc.mu.Unlock()
			err := gc.Viper.MergeConfig(data)
			if err != nil {
				if gc.editTimes == 0 {
					log.Fatal(err)
				} else {
					log.Error(err)
					return err
				}
			}
			return nil
		}, func() error {
			gc.mu.TryLock()
			defer gc.mu.Unlock()
			gc.inject(gc.Config, gc.Dao)
			gc.editTimes++
			return nil
		})
		if err != nil {
			return
		}
		gc.defers = append(gc.defers, func() {
			if err := localConfig.Close(); err != nil {
				log.Errorf("close local config error: %v", err)
			}
		})
	}

	if cfgcenter != nil {
		err = cfgcenter.Handle(func(data io.Reader) error {
			gc.mu.TryLock()
			defer gc.mu.Unlock()
			err := gc.Viper.MergeConfig(data)
			if err != nil {
				if gc.editTimes == 0 {
					log.Fatal(err)
				} else {
					log.Error(err)
					return err
				}
			}
			gc.inject(gc.Config, gc.Dao)
			gc.editTimes++
			return nil
		})
		if err != nil {
			log.Fatalf("config error: %v", err)
		}
	} else {
		gc.inject(gc.Config, gc.Dao)
	}
}

func (gc *globalConfig[C, D]) beforeInjectCall(conf Config, dao Dao) {
	conf.BeforeInject()
	if c, ok := conf.(beforeInjectWithRoot); ok {
		c.BeforeInjectWithRoot(&gc.RootConfig)
	}
	if dao != nil {
		dao.BeforeInject()
		if c, ok := dao.(beforeInjectWithRoot); ok {
			c.BeforeInjectWithRoot(&gc.RootConfig)
		}
	}
}

func (gc *globalConfig[C, D]) Defer(deferf ...func()) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.defers = append(gc.defers, deferf...)
}

func closeDao(dao Dao) error {
	var errs multierr.MultiError
	daoValue := reflect.ValueOf(dao)
	if daoValue.Kind() != reflect.Pointer {
		daoValue = daoValue.Elem()
	}
	for i := range daoValue.NumField() {
		fieldV := daoValue.Field(i)
		if fieldV.Type().Kind() == reflect.Struct {
			fieldV = daoValue.Field(i).Addr()
		}
		if !fieldV.IsValid() || fieldV.IsNil() {
			continue
		}
		inter := fieldV.Interface()
		if daofield, ok := inter.(daopkg.DaoField); ok {
			if err := daofield.Close(); err != nil {
				errs.Append(err)
			}

		}
	}
	return errs.Error()
}
