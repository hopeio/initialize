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

// App is the central container that holds root config, builtin config,
// user-defined Config and Dao, and drives the full initialization lifecycle.
//
// Concurrency model (snapshot replace): confSnapshot is the sole storage for
// config; Conf() is the sole read path. Hot reload builds a fresh instance,
// runs the full inject lifecycle, then atomically swaps confSnapshot.
// Published snapshots are immutable. Callers that need startup-time values
// should read Conf() once during startup and keep their own copy; the library
// does not retain a first-generation reference.
type App[C any, D any, CPtr ConfigPtr[C], DPtr DaoPtr[D]] struct {
	RootConfig    RootConfig `mapstructure:",squash"`
	BuiltinConfig builtinConfig

	Dao DPtr

	*viper.Viper
	// confSnapshot is the current config snapshot; replaced wholesale after a
	// successful hot reload so readers need no lock.
	confSnapshot atomic.Pointer[C]
	editTimes    uint32
	defers       []func()
	// configCenters holds the centers passed to Start, keyed by Type(). They are
	// registered before assembly and resolved by [ConfigCenter].Type; all access
	// happens on the Start call stack, so no lock is needed.
	configCenters map[string]ConfigCenter
	// started is the single lifecycle flag: true once Start has begun. It guards
	// Start against re-entry, switches the file/center watchers from
	// merge-only to snapshot replacement, and gates Inject.
	started atomic.Bool
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
// (i.e. before New returns).
func (a *App[C, D, CPtr, DPtr]) Conf() CPtr {
	return CPtr(a.confSnapshot.Load())
}

// newApp allocates a Global with sane defaults and a fresh Viper instance.
func newApp[C any, D any, CPtr ConfigPtr[C], DPtr DaoPtr[D]]() *App[C, D, CPtr, DPtr] {
	a := &App[C, D, CPtr, DPtr]{
		RootConfig: RootConfig{
			EnvConfig: EnvConfig{Debug: true},
		},
		Viper: viper.NewWithOptions(viper.WithCodecRegistry(codecRegistry)),
	}
	return a
}

// New allocates an App with zero-value Config and Dao instances. It is a pure
// constructor: no config file is read, no flag parsed, no watcher started, no
// client connected. Everything deferred to Start, which main must call before
// touching Conf or Dao:
//
//	var app = initialize.New[config, dao]()
//	func main() { defer app.Start()(); ... }
//
// Custom config centers are passed as Start arguments (registered on the app)
// Start; the center named in the config's [ConfigCenter].Type is picked up
// during assembly.
func New[C any, D any, CPtr ConfigPtr[C], DPtr DaoPtr[D]]() *App[C, D, CPtr, DPtr] {
	app := newApp[C, D, CPtr, DPtr]()
	app.Dao = DPtr(new(D))
	return app
}

// NewConfig is a shortcut for applications that only need a Config (no custom Dao).
func NewConfig[C any, CPtr ConfigPtr[C]]() *App[C, EmbeddedPresets, CPtr, *EmbeddedPresets] {
	return New[C, EmbeddedPresets, CPtr, *EmbeddedPresets]()
}

// boot runs the assembly phase of Start: root config defaults, flag/env binding,
// config center registration, file watchers and the first config injection. It
// performs file and (if configured) config-center I/O but never connects the dao.
func (a *App[C, D, CPtr, DPtr]) boot(conf CPtr) {
	applyTagDefaults(&a.RootConfig)
	a.applyFlagConfig("", &a.RootConfig)
	a.RootConfig.AfterInject()

	a.defers = append(a.defers, func() {
		if err := closeDao(a.Dao); err != nil {
			log.Errorf("close Dao error: %v", err)
		}
	})
	a.loadConfig(conf)
}

// var app = initialize.New[config, dao]()
// func main() {
// 	defer app.Start()()
// }

// Start connects the configured dao (GORM/Redis/... via each DaoField's Init)
// and then runs the dao hooks that require live clients (AfterInject, e.g. OTel
// instrumentation). New only assembles: config loading, snapshot publishing and
// watchers run during package init without any network I/O, so importing a
// package that owns an App no longer connects to anything. A process that reads
// the dao must call Start first, typically from main:
//
//	func main() { defer app.Start()(); ... }
//
// Fatal on failure: a startup-phase connection error is unrecoverable, matching
// the policy of config loading. Idempotent: a second call returns the same stop
// function without reconnecting.
// Start runs the whole launch: assembly (config loading, watchers, snapshot
// publishing) followed by connecting the dao and running the hooks that need
// live clients (AfterInject, e.g. OTel instrumentation). Fatal on failure: a
// startup-phase error is unrecoverable. Idempotent: a second call returns the
// same stop function. The returned stop releases everything in reverse
// registration order — main keeps it alive with defer:
//
//	func main() { defer app.Start()(); ... }
//
// A config center, when used, is passed as a Start argument: the arguments are
// registered before assembly, and the one whose Type matches the config file's
// [ConfigCenter].Type is picked up during assembly.
func (a *App[C, D, CPtr, DPtr]) Start(centers ...ConfigCenter) func() {
	if a.started.Swap(true) {
		return a.stop
	}
	a.registerConfigCenters(centers)
	a.boot(CPtr(new(C)))
	if a.Dao != nil {
		if err := a.injectDao(a.Dao); err != nil {
			log.Fatalf("initialize: connect dao: %v", err)
		}
	}
	return a.stop
}

// stop releases all resources in reverse registration order (defers), then closes
// the config center and flushes the logger. Not exported: callers receive it from
// Start, so it is always paired with a started app.
func (a *App[C, D, CPtr, DPtr]) stop() {
	a.mu.Lock()
	defers := a.defers
	a.defers = nil
	a.mu.Unlock()
	// Call defers in reverse order (release business resources first).
	for i := len(defers) - 1; i >= 0; i-- {
		defers[i]()
	}
	if a.RootConfig.ConfigCenter.ConfigCenter != nil {
		if err := a.RootConfig.ConfigCenter.ConfigCenter.Close(); err != nil {
			log.Errorf("close config center error: %v", err)
		}
	}
	log.Sync()
}

const defaultConfigName = "config"

// loadConfig discovers and reads the config file, sets up file watching,
// connects to the config center, and performs the initial inject into conf.
func (a *App[C, D, CPtr, DPtr]) loadConfig(conf CPtr) {
	executable, err := os.Executable()
	if err != nil {
		log.Fatalf("get executable error: %v", err)
	}
	a.RootConfig.Executable = executable
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("get work dir error: %v", err)
	}
	a.RootConfig.ExecDir = wd
	a.Viper.AutomaticEnv()
	var format string
	// find config
	if a.RootConfig.ConfPath == "" {
		log.Debugf("lack of flag -c or --config, searching 'config.*' in %s or %s", wd, wd+fs.PathSeparator+"config")
		for _, ext := range viper.SupportedExts {
			filePath := filepath.Join(".", defaultConfigName+"."+ext)
			if fs.Exist(filePath) {
				log.Debugf("found file: '%s'", filePath)
				a.RootConfig.ConfPath = filePath
				format = ext
				break
			}
		}
		// find in config dir
		if a.RootConfig.ConfPath == "" {
			configDir := filepath.Join(wd, "config")
			fileInfo, err := os.Stat(configDir)
			if err == nil && fileInfo.IsDir() {
				for _, ext := range viper.SupportedExts {
					filePath := filepath.Join(configDir, defaultConfigName+"."+ext)
					if fs.Exist(filePath) {
						log.Debugf("found file: '%s'", filePath)
						a.RootConfig.ConfPath = filePath
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
	if a.RootConfig.ConfPath != "" {
		a.RootConfig.ConfPath, err = filepath.Abs(a.RootConfig.ConfPath)
		if err != nil {
			log.Fatalf("get abs path error: %v", err)
		}
		log.Infof("load config from: '%s'", a.RootConfig.ConfPath)
		if format == "" {
			format = filepath.Ext(a.RootConfig.ConfPath)
			if format != "" {
				// remove .
				format = format[1:]
				if !slices.Contains(viper.SupportedExts, format) {
					log.Fatalf("unsupport config format:%s, support: %v", format, viper.SupportedExts)
				}
			} else {
				log.Fatalf("config path:%s need format ext, support: %v", a.RootConfig.ConfPath, viper.SupportedExts)
			}
		}

		a.RootConfig.ConfigCenter.Format = format
		a.Viper.SetConfigType(format)
		a.Viper.SetConfigFile(a.RootConfig.ConfPath)
		err := a.Viper.ReadInConfig()
		if err != nil {
			log.Fatal(err)
		}
	}

	a.setRootConfig()
	a.setEnvConfig()

	var singleTemplateFileConfig bool
	if a.RootConfig.EnvConfig.ConfigCenter.ConfigCenter == nil {
		if a.RootConfig.Env == "" {
			singleTemplateFileConfig = true
		}
		if a.RootConfig.ConfigCenter.Type != "" {
			a.RootConfig.ConfigCenter.ConfigCenter = a.getCenter(a.RootConfig.ConfigCenter.Type)
		}
	}
	cfgcenter := a.RootConfig.ConfigCenter.ConfigCenter
	if cfgcenter != nil {
		flagPrefix := strings.Join([]string{"configcenter", strings.ToLower(a.RootConfig.ConfigCenter.Type)}, ".")
		if a.RootConfig.Name != "" {
			flagPrefix = strings.ToLower(a.RootConfig.Name) + "." + flagPrefix
		}
		a.applyFlagConfig(flagPrefix, cfgcenter)
	}
	// hook function
	a.beforeInjectCall(conf, a.Dao)
	a.genConfigTemplate(conf, singleTemplateFileConfig)
	localConfig := &a.RootConfig.LocalConfig
	if a.RootConfig.Env != "" {
		var defaultEnvConfigPath string
		if a.RootConfig.ConfPath != "" {
			defaultEnvConfigPath = a.RootConfig.ConfPath[:len(a.RootConfig.ConfPath)-len(filepath.Ext(a.RootConfig.ConfPath))] + "." + a.RootConfig.Env + "." + a.RootConfig.ConfigCenter.Format
		} else if a.RootConfig.ConfigCenter.Format != "" {
			a.Viper.SetConfigType(a.RootConfig.ConfigCenter.Format)
			defaultEnvConfigPath = defaultConfigName + "." + a.RootConfig.Env + "." + a.RootConfig.ConfigCenter.Format
		} else {
			for _, ext := range viper.SupportedExts {
				filePath := filepath.Join(".", defaultConfigName+"."+a.RootConfig.Env+"."+ext)
				if fs.Exist(filePath) {
					defaultEnvConfigPath = filePath
					log.Debugf("found file: '%s'", filePath)
					a.RootConfig.ConfigCenter.Format = ext
					a.Viper.SetConfigType(ext)
					break
				}
			}
		}

		if defaultEnvConfigPath != "" && fs.Exist(defaultEnvConfigPath) && !slices.Contains(localConfig.Paths, defaultEnvConfigPath) {
			localConfig.Paths = append([]string{defaultEnvConfigPath}, localConfig.Paths...)
		}
	}

	merge := func(data io.Reader) error {
		a.mu.Lock()
		defer a.mu.Unlock()
		err := a.Viper.MergeConfig(data)
		if err != nil {
			log.Fatal(err)
		}
		return nil
	}
	onChange := func(data io.Reader) error {
		a.mu.Lock()
		err := a.Viper.MergeConfig(data)
		a.mu.Unlock()
		if err != nil {
			log.Error(err)
			return err
		}
		// Before initial inject finishes, only merge; the first inject will
		// read the merged data. After that, replace the snapshot.
		if a.started.Load() {
			a.reloadConfig()
		}
		return nil
	}

	if len(localConfig.Paths) > 0 {
		err = localConfig.Handle(context.Background(), merge, onChange)
		if err != nil {
			log.Fatal(err)
			return
		}
		a.defers = append(a.defers, func() {
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
	// Assembly only: the dao gets its configuration here but clients are built
	// later by Start, so package init performs zero network I/O.
	a.reloadMu.Lock()
	defer a.reloadMu.Unlock()
	if err := a.injectConfig(conf, a.Dao); err == nil {
		a.confSnapshot.Store(conf)
	}
}

// reloadConfig applies a hot reload via snapshot replacement: it builds a brand-new
// Config instance, runs the full injection lifecycle on it, and atomically publishes
// it on success. Readers holding the old snapshot keep a consistent view; a failed
// reload leaves the current snapshot untouched.
func (a *App[C, D, CPtr, DPtr]) reloadConfig() {
	a.reloadMu.Lock()
	defer a.reloadMu.Unlock()
	newConf := CPtr(new(C))
	a.beforeInjectCall(newConf, nil)
	if err := a.injectConfig(newConf, nil); err != nil {
		return
	}
	a.confSnapshot.Store(newConf)
	a.editTimes++
}

// beforeInjectCall invokes BeforeInject and BeforeInjectWithRoot on the given conf and dao.
func (a *App[C, D, CPtr, DPtr]) beforeInjectCall(conf Config, dao Dao) {
	conf.BeforeInject()
	if c, ok := conf.(beforeInjectWithRoot); ok {
		c.BeforeInjectWithRoot(&a.RootConfig)
	}
	if dao != nil {
		dao.BeforeInject()
		if c, ok := dao.(beforeInjectWithRoot); ok {
			c.BeforeInjectWithRoot(&a.RootConfig)
		}
	}
}

// Defer registers cleanup functions that will be called in reverse order during Cleanup.
func (a *App[C, D, CPtr, DPtr]) Defer(deferf ...func()) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.defers = append(a.defers, deferf...)
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
