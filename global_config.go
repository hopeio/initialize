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

// ConfigPtr constrains CPtr to be *C and implement Config,
// letting the type system guarantee "must be a pointer type" at compile time.
type ConfigPtr[T any] interface {
	*T
	Config
}

// DaoPtr constrains DPtr to be *D and implement Dao.
type DaoPtr[T any] interface {
	*T
	Dao
}

// globalConfig is the central container that holds root config, builtin config,
// user-defined Config and Dao, and drives the full initialization lifecycle.
//
// Concurrency model (snapshot replace): confSnapshot is the sole storage for
// config; Conf() is the sole read path. Hot reload builds a fresh instance,
// runs the full inject lifecycle, then atomically swaps confSnapshot.
// Published snapshots are immutable. Callers that need startup-time values
// should read Conf() once during startup and keep their own copy; the library
// does not retain a first-generation reference.
type globalConfig[C any, D any, CPtr ConfigPtr[C], DPtr DaoPtr[D]] struct {
	RootConfig    RootConfig `mapstructure:",squash"`
	BuiltinConfig builtinConfig

	Dao DPtr

	*viper.Viper
	// confSnapshot is the current config snapshot; replaced wholesale after a
	// successful hot reload so readers need no lock.
	confSnapshot atomic.Pointer[C]
	editTimes    uint32
	defers       []func()
	initialized  atomic.Bool
	// Lock nesting is reloadMu → mu only (never the reverse):
	// mu is the inner data lock for short Viper/defers critical sections
	// (microseconds). Never run user hooks under mu or Defer inside a hook
	// deadlocks.
	// reloadMu is the outer flow lock serializing initial inject, hot reload,
	// and extra inject (may take seconds, e.g. injectDao connecting). It keeps
	// Viper reads through snapshot publish from overlapping concurrent reloads
	// so an older snapshot cannot overwrite a newer one.
	mu       sync.Mutex
	reloadMu sync.Mutex
}

// Conf returns the current config snapshot without locking. After a hot reload it
// returns the newly built snapshot. Published snapshots are immutable: never write
// to the returned value. It returns nil only before the initial injection completes
// (i.e. before NewGlobal returns).
func (gc *globalConfig[C, D, CPtr, DPtr]) Conf() CPtr {
	return CPtr(gc.confSnapshot.Load())
}

// newGlobal allocates a globalConfig with sane defaults and a fresh Viper instance.
func newGlobal[C any, D any, CPtr ConfigPtr[C], DPtr DaoPtr[D]]() *globalConfig[C, D, CPtr, DPtr] {
	gc := &globalConfig[C, D, CPtr, DPtr]{
		RootConfig: RootConfig{
			EnvConfig: EnvConfig{Debug: true},
		},
		Viper: viper.NewWithOptions(viper.WithCodecRegistry(codecRegistry)),
	}
	return gc
}

// NewGlobal creates a globalConfig by allocating zero-value instances of C and D,
// then runs the full initialization sequence. The library owns both instances:
// config is read exclusively through Conf(), so callers can never hold on to a
// stale pre-reload pointer; defaults belong in the BeforeInject hook.
// var Global = initialize.NewGlobal[config, dao]()
func NewGlobal[C any, D any, CPtr ConfigPtr[C], DPtr DaoPtr[D]](configCenter ...ConfigCenter) *globalConfig[C, D, CPtr, DPtr] {
	gc := newGlobal[C, D, CPtr, DPtr]()
	gc.Dao = DPtr(new(D))
	gc.init(CPtr(new(C)), configCenter...)
	return gc
}

// NewGlobalConfig is a shortcut for applications that only need a Config (no custom Dao).
func NewGlobalConfig[C any, CPtr ConfigPtr[C]](configCenter ...ConfigCenter) *globalConfig[C, EmbeddedPresets, CPtr, *EmbeddedPresets] {
	return NewGlobal[C, EmbeddedPresets, CPtr, *EmbeddedPresets](configCenter...)
}

// init registers config centers, loads the config file, and performs the first injection
// into conf.
func (gc *globalConfig[C, D, CPtr, DPtr]) init(conf CPtr, configCenter ...ConfigCenter) {
	applyTagDefaults(&gc.RootConfig)
	gc.applyFlagConfig("", &gc.RootConfig)
	gc.RootConfig.AfterInject()
	// Pluggable config centers keep the core dependency-light; serialization
	// may follow the same pattern later. Register centers here (local file is
	// registered by default elsewhere).
	for _, cc := range configCenter {
		RegisterConfigCenter(cc)
	}

	gc.defers = append(gc.defers, func() {
		if err := closeDao(gc.Dao); err != nil {
			log.Errorf("close Dao error: %v", err)
		}
	})
	gc.loadConfig(conf)
	gc.initialized.Store(true)
}

// var Global = initialize.NewGlobal[C,D]()
// func main(){
// 		defer global.Global.Cleanup()
// }

// Cleanup releases all resources in reverse registration order (defers), then closes the
// config center and flushes the logger. Call via defer after NewGlobal.
func (gc *globalConfig[C, D, CPtr, DPtr]) Cleanup() {
	gc.mu.Lock()
	defers := gc.defers
	gc.defers = nil
	gc.mu.Unlock()
	// Call defers in reverse order (release business resources first).
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
// connects to the config center, and performs the initial inject into conf.
func (gc *globalConfig[C, D, CPtr, DPtr]) loadConfig(conf CPtr) {
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
	gc.beforeInjectCall(conf, gc.Dao)
	gc.genConfigTemplate(conf, singleTemplateFileConfig)
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
		// Before initial inject finishes, only merge; the first inject will
		// read the merged data. After that, replace the snapshot.
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
		// Config center is closed once in Cleanup (after business defers);
		// do not register Close here to avoid a double Close.
	}
	// Watchers are running; serialize initial inject and hot reload with reloadMu.
	gc.reloadMu.Lock()
	defer gc.reloadMu.Unlock()
	if err := gc.inject(conf, gc.Dao); err == nil {
		gc.confSnapshot.Store(conf)
	}
}

// reloadConfig applies a hot reload via snapshot replacement: it builds a brand-new
// Config instance, runs the full injection lifecycle on it, and atomically publishes
// it on success. Readers holding the old snapshot keep a consistent view; a failed
// reload leaves the current snapshot untouched.
func (gc *globalConfig[C, D, CPtr, DPtr]) reloadConfig() {
	gc.reloadMu.Lock()
	defer gc.reloadMu.Unlock()
	newConf := CPtr(new(C))
	gc.beforeInjectCall(newConf, nil)
	if err := gc.inject(newConf, nil); err != nil {
		return
	}
	gc.confSnapshot.Store(newConf)
	gc.editTimes++
}

// beforeInjectCall invokes BeforeInject and BeforeInjectWithRoot on the given conf and dao.
func (gc *globalConfig[C, D, CPtr, DPtr]) beforeInjectCall(conf Config, dao Dao) {
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
func (gc *globalConfig[C, D, CPtr, DPtr]) Defer(deferf ...func()) {
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
		// Unexported fields cannot Interface(); IsNil is only valid for nillable kinds.
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
