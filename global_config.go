package initialize

import (
	"github.com/hopeio/initialize/conf_center"
	"github.com/hopeio/initialize/conf_center/local"
	"github.com/hopeio/initialize/conf_dao"
	"github.com/hopeio/initialize/rootconf"
	"github.com/hopeio/utils/errors/multierr"
	"github.com/hopeio/utils/io/fs"
	"github.com/hopeio/utils/slices"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/hopeio/utils/log"
)

// 约定大于配置
var (
	gConfig = &globalConfig{
		RootConfig: rootconf.RootConfig{
			ConfPath:  "",
			EnvConfig: rootconf.EnvConfig{Debug: true},
		},

		Viper: viper.New(),
		lock:  sync.RWMutex{},
	}
	decoderConfigOptions = []viper.DecoderConfigOption{
		viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.TextUnmarshallerHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		)),
		func(config *mapstructure.DecoderConfig) {
			config.Squash = true
		},
	}
)

func GlobalConfig() *globalConfig {
	return gConfig
}

// globalConfig
// 全局配置
type globalConfig struct {
	RootConfig rootconf.RootConfig `mapstructure:",squash"`
	BuiltinConfig

	conf Config
	dao  Dao

	*viper.Viper
	/*
		cacheConf      any*/
	editTimes   uint32
	defers      []func()
	initialized bool
	lock        sync.RWMutex

	// 为后续仍有需要注入的config和dao保留的后门,与Inject(Config,Dao) 配合
	injectConfs []Config
	injectDaos  []Dao
}

func Start(conf Config, dao Dao, configCenter ...conf_center.ConfigCenter) func() {
	if gConfig.initialized {
		return func() {}
	}

	if reflect.ValueOf(conf).IsNil() {
		log.Fatalf("初始化错误: 配置不能为空")
	}

	// 为支持自定义配置中心,并且遵循依赖最小化原则,配置中心改为可插拔的,考虑将配置序列话也照此重做
	// 注册配置中心,默认注册本地文件
	conf_center.RegisterConfigCenter(local.ConfigCenter)
	for _, cc := range configCenter {
		conf_center.RegisterConfigCenter(cc)
	}

	gConfig.setConfDao(conf, dao)
	gConfig.loadConfig()
	gConfig.initialized = true
	return func() {
		// 倒序调用defer
		for i := len(gConfig.defers) - 1; i > 0; i-- {
			gConfig.defers[i]()
		}
		if gConfig.RootConfig.ConfigCenter.ConfigCenter != nil {
			if err := gConfig.RootConfig.ConfigCenter.ConfigCenter.Close(); err != nil {
				log.Errorf("close config center error: %v", err)
			}
		}
		log.Sync()
	}
}

func (gc *globalConfig) setConfDao(conf Config, dao Dao) {
	if !gc.initialized {
		gc.conf = conf
		gc.dao = dao
	} else {
		gc.injectConfs = append(gc.injectConfs, conf)
		gc.injectDaos = append(gc.injectDaos, dao)
	}

	if dao != nil {
		gc.defers = append(gc.defers, func() {
			if err := closeDao(dao); err != nil {
				log.Errorf("close dao error: %v", err)
			}
		})
	}

}

const defaultConfigName = "config"

func (gc *globalConfig) loadConfig() {
	gc.Viper.AutomaticEnv()
	var format string
	// find config
	if gc.RootConfig.ConfPath == "" {
		log.Debug("searching for config in .")
		for _, ext := range viper.SupportedExts {
			filePath := filepath.Join(".", defaultConfigName+"."+ext)
			if b := fs.Exist(filePath); b {
				log.Debug("found file", "file", filePath)
				gc.RootConfig.ConfPath = filePath
				format = ext
				break
			}
		}
		if format == "" {
			log.Fatal("not found config")
		}
	}
	if gc.RootConfig.ConfPath != "" {
		log.Infof("load config from: %s", gc.RootConfig.ConfPath)
		if format == "" {
			format = path.Ext(gc.RootConfig.ConfPath)
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

	gc.setBasicConfig()
	gc.setEnvConfig()
	for i := range gc.RootConfig.NoInject {
		gc.RootConfig.NoInject[i] = strings.ToUpper(gc.RootConfig.NoInject[i])
	}

	var singleTemplateFileConfig bool
	if gc.RootConfig.EnvConfig.ConfigCenter.ConfigCenter == nil {
		if gc.RootConfig.Env == "" {
			singleTemplateFileConfig = true
		}
		// 单配置文件
		gc.RootConfig.ConfigCenter.ConfigCenter = &local.Local{
			Conf: local.Config{
				ConfigPath: gc.RootConfig.ConfPath,
			},
		}
		applyFlagConfig(gc.Viper, gc.RootConfig.ConfigCenter.ConfigCenter)
	}

	// hook function
	gc.beforeInjectCall(gc.conf, gc.dao)
	gc.genConfigTemplate(singleTemplateFileConfig)

	cfgcenter := gc.RootConfig.ConfigCenter.ConfigCenter
	err := cfgcenter.Handle(gc.UnmarshalAndSet)
	if err != nil {
		log.Fatalf("配置错误: %v", err)
	}
}

func (gc *globalConfig) beforeInjectCall(conf Config, dao Dao) {
	conf.BeforeInject()
	if c, ok := conf.(BeforeInjectWithRoot); ok {
		c.BeforeInjectWithRoot(&gc.RootConfig)
	}
	if dao != nil {
		dao.BeforeInject()
		if c, ok := dao.(BeforeInjectWithRoot); ok {
			c.BeforeInjectWithRoot(&gc.RootConfig)
		}
	}
}

func (gc *globalConfig) DeferFunc(deferf ...func()) {
	gc.lock.Lock()
	defer gc.lock.Unlock()
	gc.defers = append(gc.defers, deferf...)
}

func RegisterDeferFunc(deferf ...func()) {
	gConfig.lock.Lock()
	defer gConfig.lock.Unlock()
	gConfig.defers = append(gConfig.defers, deferf...)
}

func closeDao(dao Dao) error {
	var errs multierr.MultiError
	daoValue := reflect.ValueOf(dao).Elem()
	for i := 0; i < daoValue.NumField(); i++ {
		fieldV := daoValue.Field(i)
		if fieldV.Type().Kind() == reflect.Struct {
			fieldV = daoValue.Field(i).Addr()
		}
		if !fieldV.IsValid() || fieldV.IsNil() {
			continue
		}
		inter := fieldV.Interface()
		if daofield, ok := inter.(conf_dao.DaoField); ok {
			if err := daofield.Close(); err != nil {
				errs.Append(err)
			}

		}
	}

	if errs.HasErrors() {
		return &errs
	}
	return nil
}
