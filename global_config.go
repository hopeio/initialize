/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/hopeio/gox/log"
	"github.com/hopeio/gox/os/fs"
	"github.com/spf13/viper"
	"go.uber.org/multierr"
)

// globalConfig is the central container that holds root config, builtin config,
// user-defined Config and Dao, and drives the full initialization lifecycle.
//
// 并发模型（快照替换）：Config 字段是启动时注入的快照，初始化完成后不再被写入；
// 热更新会构建一个全新的 C 实例，走完整注入生命周期后原子替换 confSnapshot。
// 读方通过 Conf() 无锁获取当前快照，快照一经发布即不可变。
type globalConfig[C Config, D Dao] struct {
	RootConfig    RootConfig `mapstructure:",squash"`
	BuiltinConfig builtinConfig

	// conf 是启动快照：初始注入完成后不再写入。
	// 刻意不导出：外部只能经 Conf()（当前快照）或 StartupConf()（启动快照）访问，
	// 避免调用方误把启动快照当作能感知热更新的实时配置。
	conf C
	Dao  D

	*viper.Viper
	// 当前配置快照（存 C），热更新成功后整体替换，读方零锁
	confSnapshot atomic.Value
	editTimes    uint32
	defers       []func()
	initialized  atomic.Bool
	// mu 只保护 Viper 操作与 defers 的短临界区，锁内不执行任何用户 hook
	mu sync.RWMutex
	// reloadMu 串行化初始注入、热更新与追加注入
	reloadMu sync.Mutex
}

// Conf returns the current config snapshot without locking. After a hot reload it
// returns the newly built snapshot. Published snapshots are immutable: never write
// to the returned value. This is the default accessor for reading config.
func (gc *globalConfig[C, D]) Conf() C {
	if v := gc.confSnapshot.Load(); v != nil {
		return v.(C)
	}
	return gc.conf
}

// StartupConf returns the startup snapshot: the Config instance injected at boot,
// which never observes hot reloads. Use it only when values must stay consistent
// with what the process actually started with (e.g. the listen address);
// otherwise prefer Conf.
func (gc *globalConfig[C, D]) StartupConf() C {
	return gc.conf
}

// newGlobal allocates a globalConfig with sane defaults and a fresh Viper instance.
func newGlobal[C Config, D Dao]() *globalConfig[C, D] {
	gc := &globalConfig[C, D]{
		RootConfig: RootConfig{
			EnvConfig: EnvConfig{Debug: true},
		},
		Viper: viper.NewWithOptions(viper.WithCodecRegistry(codecRegistry)),
	}
	return gc
}

// NewGlobalWith creates a globalConfig with the provided Config and Dao instances and
// immediately runs the full initialization sequence.
func NewGlobalWith[C Config, D Dao](conf C, dao D, configCenter ...ConfigCenter) *globalConfig[C, D] {
	gc := newGlobal[C, D]()
	gc.conf = conf
	gc.Dao = dao
	gc.init(configCenter...)
	return gc
}

// NewGlobal creates a globalConfig by allocating zero-value instances of C and D via reflection,
// then runs the full initialization sequence.
// var Global = initialize.NewGlobal[C,D]()
func NewGlobal[C Config, D Dao](configCenter ...ConfigCenter) *globalConfig[C, D] {
	gc := newGlobal[C, D]()
	v := reflect.ValueOf(&gc.conf).Elem()
	if v.Kind() == reflect.Struct {
		log.Fatalf("generic type should be a pointer type")
	}
	v.Set(reflect.New(reflect.TypeOf(gc.conf).Elem()))
	v = reflect.ValueOf(&gc.Dao).Elem()
	if v.Kind() == reflect.Struct {
		log.Fatalf("generic type should be a pointer type")
	}
	v.Set(reflect.New(reflect.TypeOf(gc.Dao).Elem()))
	gc.init(configCenter...)
	return gc
}

// Start is a convenience wrapper around NewGlobalWith that returns only the Cleanup function.
func Start[C Config, D Dao](conf C, dao D, configCenter ...ConfigCenter) func() {
	gc := NewGlobalWith(conf, dao, configCenter...)
	return gc.Cleanup
}

// NewGlobalConfig is a shortcut for applications that only need a Config (no custom Dao).
func NewGlobalConfig[C Config](configCenter ...ConfigCenter) *globalConfig[C, *EmbeddedPresets] {
	return NewGlobal[C, *EmbeddedPresets](configCenter...)
}

// init registers config centers, loads the config file, and performs the first injection.
func (gc *globalConfig[C, D]) init(configCenter ...ConfigCenter) {
	gc.applyFlagConfig("", &gc.RootConfig)
	gc.RootConfig.AfterInject()
	// 为支持自定义配置中心,并且遵循依赖最小化原则,配置中心改为可插拔的,考虑将配置序列话也照此重做
	// 注册配置中心,默认注册本地文件
	for _, cc := range configCenter {
		RegisterConfigCenter(cc)
	}

	gc.defers = append(gc.defers, func() {
		if err := closeDao(gc.Dao); err != nil {
			log.Errorf("close Dao error: %v", err)
		}
	})
	gc.loadConfig()
	gc.initialized.Store(true)
}

// var Global = initialize.NewGlobal[C,D]()
// func main(){
// 		defer global.Global.Cleanup()
// }

// Cleanup releases all resources in reverse registration order (defers), then closes the
// config center and flushes the logger. Call via defer after NewGlobal/NewGlobalWith.
func (gc *globalConfig[C, D]) Cleanup() {
	gc.mu.Lock()
	defers := gc.defers
	gc.defers = nil
	gc.mu.Unlock()
	// 倒序调用defer（先释放业务资源）
	for i := len(defers) - 1; i >= 0; i-- {
		defers[i]()
	}
	if gc.RootConfig.ConfigCenter.ConfigCenter != nil {
		if err := gc.RootConfig.ConfigCenter.ConfigCenter.Close(); err != nil {
			log.Errorf("close config center error: %v", err)
		}
	}
	log.Sync()
}

const defaultConfigName = "config"

// loadConfig discovers and reads the config file, sets up file watching,
// connects to the config center, and performs the initial inject.
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
		log.Debugf("lack of flag -c or --config, searching 'config.*' in %s or %s", wd, wd+fs.PathSeparator+"config")
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
					log.Fatalf("unsupport config format:%s, support: %v", format, viper.SupportedExts)
				}
			} else {
				log.Fatalf("config path:%s need format ext, support: %v", gc.RootConfig.ConfPath, viper.SupportedExts)
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

	var singleTemplateFileConfig bool
	if gc.RootConfig.EnvConfig.ConfigCenter.ConfigCenter == nil {
		if gc.RootConfig.Env == "" {
			singleTemplateFileConfig = true
		}
		if gc.RootConfig.ConfigCenter.Type != "" {
			gc.RootConfig.ConfigCenter.ConfigCenter = GetConfigCenter(gc.RootConfig.ConfigCenter.Type)
		}
	}
	cfgcenter := gc.RootConfig.ConfigCenter.ConfigCenter
	if cfgcenter != nil {
		flagPrefix := strings.Join([]string{"configcenter", strings.ToLower(gc.RootConfig.ConfigCenter.Type)}, ".")
		if gc.RootConfig.Name != "" {
			flagPrefix = strings.ToLower(gc.RootConfig.Name) + "." + flagPrefix
		}
		gc.applyFlagConfig(flagPrefix, cfgcenter)
	}
	// hook function
	gc.beforeInjectCall(gc.conf, gc.Dao)
	gc.genConfigTemplate(singleTemplateFileConfig)
	localConfig := &gc.RootConfig.LocalConfig
	if gc.RootConfig.Env != "" {
		var defaultEnvConfigPath string
		if gc.RootConfig.ConfPath != "" {
			defaultEnvConfigPath = gc.RootConfig.ConfPath[:len(gc.RootConfig.ConfPath)-len(filepath.Ext(gc.RootConfig.ConfPath))] + "." + gc.RootConfig.Env + "." + gc.RootConfig.ConfigCenter.Format
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

	merge := func(data io.Reader) error {
		gc.mu.Lock()
		defer gc.mu.Unlock()
		err := gc.Viper.MergeConfig(data)
		if err != nil {
			log.Fatal(err)
		}
		return nil
	}
	onChange := func(data io.Reader) error {
		gc.mu.Lock()
		err := gc.Viper.MergeConfig(data)
		gc.mu.Unlock()
		if err != nil {
			log.Error(err)
			return err
		}
		// 初始注入完成前只合并数据，首次 inject 时自然读到；之后走快照替换
		if gc.initialized.Load() {
			gc.reloadConfig()
		}
		return nil
	}

	if len(localConfig.Paths) > 0 {
		err = localConfig.Handle(context.Background(), merge, onChange)
		if err != nil {
			log.Fatal(err)
			return
		}
		gc.defers = append(gc.defers, func() {
			if err := localConfig.Close(); err != nil {
				log.Errorf("close local config error: %v", err)
			}
		})
	}

	if cfgcenter != nil {
		err = cfgcenter.Handle(context.Background(), merge, onChange)
		if err != nil {
			log.Fatalf("config error: %v", err)
		}
		// 配置中心由 Cleanup 统一关闭（业务 defers 之后），不在此重复注册，避免双重 Close
	}
	// watcher 已启动，初始注入与热更新用 reloadMu 串行
	gc.reloadMu.Lock()
	defer gc.reloadMu.Unlock()
	if err := gc.inject(gc.conf, gc.Dao); err == nil {
		gc.confSnapshot.Store(gc.conf)
	}
}

// reloadConfig applies a hot reload via snapshot replacement: it builds a brand-new
// Config instance, runs the full injection lifecycle on it, and atomically publishes
// it on success. Readers holding the old snapshot keep a consistent view; a failed
// reload leaves the current snapshot untouched.
func (gc *globalConfig[C, D]) reloadConfig() {
	gc.reloadMu.Lock()
	defer gc.reloadMu.Unlock()
	newConf := reflect.New(reflect.TypeOf(gc.conf).Elem()).Interface().(C)
	gc.beforeInjectCall(newConf, nil)
	if err := gc.inject(newConf, nil); err != nil {
		return
	}
	gc.confSnapshot.Store(newConf)
	gc.editTimes++
}

// beforeInjectCall invokes BeforeInject and BeforeInjectWithRoot on the given conf and dao.
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

// Defer registers cleanup functions that will be called in reverse order during Cleanup.
func (gc *globalConfig[C, D]) Defer(deferf ...func()) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.defers = append(gc.defers, deferf...)
}

// closeDao closes all DaoField instances within the Dao struct, collecting errors with multierr.
func closeDao(dao Dao) error {
	var errs error
	daoValue := reflect.ValueOf(dao)
	if daoValue.Kind() == reflect.Pointer {
		if daoValue.IsNil() {
			return nil
		}
		daoValue = daoValue.Elem()
	}
	for i := range daoValue.NumField() {
		fieldV := daoValue.Field(i)
		if fieldV.Kind() == reflect.Struct && fieldV.CanAddr() {
			fieldV = fieldV.Addr()
		}
		// 未导出字段不能 Interface；IsNil 只对可为 nil 的 kind 合法
		if !fieldV.IsValid() || !fieldV.CanInterface() {
			continue
		}
		switch fieldV.Kind() {
		case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
			if fieldV.IsNil() {
				continue
			}
		}
		if daofield, ok := fieldV.Interface().(DaoField); ok {
			if err := daofield.Close(); err != nil {
				errs = multierr.Append(errs, err)
			}
		}
	}
	return errs
}
