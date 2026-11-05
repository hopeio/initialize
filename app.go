/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"context"
	"fmt"
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
	// stopped marks that cleanup already ran, so the exported Stop is idempotent
	// (Start hands out Stop itself, and a caller pairing an explicit Stop with
	// `defer app.Start()()` must not close the config center twice). It lives under
	// mu — the same lock that owns defers — rather than a second sync primitive.
	stopped bool
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
// client connected. Everything is deferred to the launch, which main must run
// before touching Conf or Dao — [App.Start] for a process entry point, or
// [App.TryStart] wherever the failure is worth handling:
//
//	var app = initialize.New[config, dao]()
//	func main() { defer app.Start()(); ... }
//
// Custom config centers are passed as start arguments (registered on the app);
// the center named in the config's [ConfigCenter].Type is picked up during
// assembly.
func New[C any, D any, CPtr ConfigPtr[C], DPtr DaoPtr[D]]() *App[C, D, CPtr, DPtr] {
	app := newApp[C, D, CPtr, DPtr]()
	app.Dao = DPtr(new(D))
	return app
}

// NewConfig is a shortcut for applications that only need a Config (no custom Dao).
func NewConfig[C any, CPtr ConfigPtr[C]]() *App[C, EmbeddedPresets, CPtr, *EmbeddedPresets] {
	return New[C, EmbeddedPresets, CPtr, *EmbeddedPresets]()
}

// boot runs the assembly phase of the launch (see [App.TryStart]): root config
// defaults, flag/env binding, file watchers and the first config injection. It
// performs file and (if configured) config-center I/O but never connects the dao.
func (a *App[C, D, CPtr, DPtr]) boot(conf CPtr) error {
	if err := applyTagDefaults(&a.RootConfig); err != nil {
		return err
	}
	if err := a.applyFlagConfig("", &a.RootConfig); err != nil {
		return err
	}
	if err := a.RootConfig.resolveLocalPaths(); err != nil {
		return err
	}

	a.defers = append(a.defers, func() {
		if err := closeDao(a.Dao); err != nil {
			log.Errorf("close Dao error: %v", err)
		}
	})
	return a.loadConfig(conf)
}

// TryStart runs the whole launch and hands the first failure back to the caller:
// assembly (config loading, watchers, snapshot publishing), then connecting the dao
// and running the hooks that need live clients (AfterInject, e.g. OTel
// instrumentation). New performs no I/O, so importing a package that owns an App
// connects to nothing; a process that reads Conf or Dao must start it first:
//
//	if err := app.TryStart(); err != nil {
//		log.Fatal(err)
//	}
//	defer app.Stop()
//
// Returning the error instead of exiting is what lets the caller pick the policy: a
// CLI subcommand can fail alone, a test can report the failure and still run the
// cases after it, and an entry point can print its own message. Use TryStart
// wherever a failure is worth reacting to — including tests, because [App.Start]
// ends the process.
//
// Cleanup has exactly one entry point, the exported [App.Stop]: a failed TryStart
// has already released everything the assembly acquired, so the deferred Stop is a
// no-op on that path and the real cleanup otherwise. Conf() stays nil until a launch
// succeeds; a failed app is not retried by another start call.
//
// Idempotent: a second successful call leaves the app as it is without reassembling
// or reconnecting. Processes with a paired lifecycle — a CLI framework's pre/post run
// hooks, or a long-running command that stops the app from a signal handler — call
// TryStart and Stop on either side instead.
//
// A config center, when used, is passed as an argument: the arguments are
// registered before assembly, and the one whose Type matches the config file's
// [ConfigCenter].Type is picked up during assembly.
func (a *App[C, D, CPtr, DPtr]) TryStart(centers ...ConfigCenter) error {
	if a.started.Swap(true) {
		return nil
	}
	return a.start(centers)
}

// Start is [App.TryStart] for a process entry point: startup assembly and the dao
// connection are unrecoverable there, so a failure logs and ends the process instead
// of forcing every main to write the same three lines. It keeps the established
// one-liner, returning [App.Stop] so the deferred call reads
//
//	func main() { defer app.Start()(); ... }
//
// It is not a shorter spelling for business code: anywhere a failure can be handled
// — a test, a subcommand of a long-running CLI, an entry point with its own exit
// message or cleanup — call [App.TryStart] and decide there. The failure still
// releases what the assembly acquired before ending the process, because Start runs
// TryStart rather than fataling from inside the config machinery.
func (a *App[C, D, CPtr, DPtr]) Start(centers ...ConfigCenter) func() {
	if err := a.TryStart(centers...); err != nil {
		log.Fatalf("initialize: %v", err)
	}
	return a.Stop
}

// start is the shared worker behind Start and TryStart: register the centers,
// assemble the config, connect the dao. It hands the first failure back so the
// entry point decides what to do about it.
func (a *App[C, D, CPtr, DPtr]) start(centers []ConfigCenter) error {
	if err := a.registerConfigCenters(centers); err != nil {
		return err
	}
	if err := a.boot(CPtr(new(C))); err != nil {
		return err
	}
	if a.Dao != nil {
		if err := a.injectDao(a.Dao); err != nil {
			return fmt.Errorf("initialize: connect dao: %w", err)
		}
	}
	return nil
}

// Stop releases what the launch acquired: the Defer callbacks in reverse registration
// order, then the config center, then log.Sync. It is the only cleanup entry point —
// TryStart hands back an error rather than a closure, so a caller never has to choose
// between two names for the same teardown — and CLI pre/post hooks or a
// signal-driven shutdown pair it with an explicit TryStart. Idempotent through the
// stopped flag under mu; calling it before a successful launch does nothing on
// purpose — marking it stopped there would turn the real cleanup into a no-op and
// leak the dao.
func (a *App[C, D, CPtr, DPtr]) Stop() {
	if !a.started.Load() {
		log.Warn("initialize: Stop before Start does nothing (cleanup belongs to a started app)")
		return
	}
	a.mu.Lock()
	if a.stopped {
		a.mu.Unlock()
		return
	}
	a.stopped = true
	a.mu.Unlock()
	a.stop()
}

// stop is Stop's worker: defers in reverse order, then the config center, then
// log.Sync. Reached only through Stop, which decides it runs at most once; user
// hooks never run under mu.
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
// Every failure goes back to Start, which releases what the assembly already
// acquired; the watchers and the center stay reachable through the app so that
// rollback can close them.
func (a *App[C, D, CPtr, DPtr]) loadConfig(conf CPtr) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("initialize: resolve executable: %w", err)
	}
	a.RootConfig.Executable = executable
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("initialize: resolve working dir: %w", err)
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
			return fmt.Errorf("initialize: resolve config path: %w", err)
		}
		log.Infof("load config from: '%s'", a.RootConfig.ConfPath)
		if format == "" {
			format = filepath.Ext(a.RootConfig.ConfPath)
			if format != "" {
				// remove .
				format = format[1:]
				if !slices.Contains(viper.SupportedExts, format) {
					return fmt.Errorf("initialize: unsupported config format %s, support: %v", format, viper.SupportedExts)
				}
			} else {
				return fmt.Errorf("initialize: config path %s needs a format extension, support: %v", a.RootConfig.ConfPath, viper.SupportedExts)
			}
		}

		a.RootConfig.ConfigCenter.Format = format
		a.Viper.SetConfigType(format)
		a.Viper.SetConfigFile(a.RootConfig.ConfPath)
		if err := a.Viper.ReadInConfig(); err != nil {
			return fmt.Errorf("initialize: read config %s: %w", a.RootConfig.ConfPath, err)
		}
	}

	if err := a.setRootConfig(); err != nil {
		return err
	}
	if err := a.setEnvConfig(); err != nil {
		return err
	}
	// Namespace implicit environment lookups now that the module name is known.
	// AutomaticEnv resolves every single-segment key straight from the environment,
	// so an unrelated ambient variable hijacks a whole config section: key "user"
	// picks up $USER (the login name), which replaces the [User] table with a string
	// and silently drops all of its values at decode time. Explicit per-field
	// bindings (init:"env:VAR") keep working, they consult VAR and NAME_VAR through
	// os.LookupEnv directly. Must run after setRootConfig, which is where Name is
	// resolved, and before any merge/inject Get.
	if envPrefix := strings.TrimSpace(a.RootConfig.Name); envPrefix != "" {
		a.Viper.SetEnvPrefix(envPrefix)
	}

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
		if err := a.applyFlagConfig(flagPrefix, cfgcenter); err != nil {
			return err
		}
	}
	// hook function
	a.beforeInjectCall(conf, a.Dao)
	if err := a.genConfigTemplate(conf, singleTemplateFileConfig); err != nil {
		return err
	}
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
		if err := a.Viper.MergeConfig(data); err != nil {
			return fmt.Errorf("initialize: merge config: %w", err)
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
		if err := localConfig.Handle(context.Background(), merge, onChange); err != nil {
			return fmt.Errorf("initialize: load local config: %w", err)
		}
		a.defers = append(a.defers, func() {
			if err := localConfig.Close(); err != nil {
				log.Errorf("close local config error: %v", err)
			}
		})
	}

	if cfgcenter != nil {
		if err := cfgcenter.Handle(context.Background(), merge, onChange); err != nil {
			return fmt.Errorf("initialize: load config center: %w", err)
		}
		// Config center is closed once in Stop (after business defers);
		// do not register Close here to avoid a double Close.
	}
	// Watchers are running; serialize initial inject and hot reload with reloadMu.
	// Assembly only: the dao gets its configuration here but clients are built
	// later by Start, so package init performs zero network I/O.
	a.reloadMu.Lock()
	defer a.reloadMu.Unlock()
	if err := a.injectConfig(conf, a.Dao); err != nil {
		return err
	}
	a.confSnapshot.Store(conf)
	return nil
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
