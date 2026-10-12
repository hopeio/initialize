/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type closingDaoField struct {
	closed bool
}

func (c *closingDaoField) Config() any { return c }

func (c *closingDaoField) Init() error { return nil }

func (c *closingDaoField) Close() error {
	c.closed = true
	return nil
}

type testDaoWithClosingField struct {
	EmbeddedPresets
	Store closingDaoField
}

func TestApp_Start_RunsDefers(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	a := NewConfig[UserConfig]()
	var deferCalled bool
	a.Defer(func() { deferCalled = true })
	a.Start()()
	if !deferCalled {
		t.Fatal("Cleanup should invoke Defer callbacks")
	}
}

func TestApp_Start_DefersReverseOrder(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	a := NewConfig[UserConfig]()
	var order []int
	a.Defer(func() { order = append(order, 1) })
	a.Defer(func() { order = append(order, 2) })
	a.Start()()
	if len(order) != 2 || order[0] != 2 || order[1] != 1 {
		t.Fatalf("defer order=%v want [2 1]", order)
	}
}

func TestCloseDao_ClosesDaoFields(t *testing.T) {
	dao := &testDaoWithClosingField{}
	if err := closeDao(dao); err != nil {
		t.Fatal(err)
	}
	if !dao.Store.closed {
		t.Fatal("closeDao should call DaoField.Close")
	}
}

func TestCloseDao_IgnoresScalarAndUnexportedFields(t *testing.T) {
	type daoMixedFields struct {
		EmbeddedPresets
		Name    string
		Count   int
		Tags    []string
		hidden  int
		Store   closingDaoField
		NilPtr  *closingDaoField
		Handler func()
	}
	d := &daoMixedFields{}
	_ = d.hidden
	// Plain / unexported / nil pointer fields used to panic in closeDao.
	if err := closeDao(d); err != nil {
		t.Fatal(err)
	}
	if !d.Store.closed {
		t.Fatal("Store should be closed")
	}
}

func TestCloseDao_AggregatesErrors(t *testing.T) {
	type errField struct {
		EmbeddedPresets
		A closingDaoField
		B errClosingField
	}
	dao := &errField{}
	err := closeDao(dao)
	if err == nil {
		t.Fatal("closeDao want aggregated error")
	}
}

type errClosingField struct{}

func (e *errClosingField) Config() any { return e }

func (e *errClosingField) Init() error { return nil }

func (e *errClosingField) Close() error { return errors.New("close err") }

// reloadTestConfig fields intentionally omit flag/env segments and avoid
// common env names like HOST/PORT so ambient environment does not interfere.
type reloadTestConfig struct {
	EmbeddedPresets
	Addr  string
	Level int
}

func TestReloadConfig_SnapshotSwap(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	a := NewConfig[reloadTestConfig]()
	a.Start()
	startup := a.Conf()
	if startup == nil {
		t.Fatal("initial snapshot should be published after Start")
	}

	if err := a.Viper.MergeConfigMap(map[string]any{"addr": "reloaded", "level": 7}); err != nil {
		t.Fatal(err)
	}
	a.reloadConfig()

	snap := a.Conf()
	if snap == startup {
		t.Fatal("reload should publish a new snapshot instance")
	}
	if snap.Addr != "reloaded" || snap.Level != 7 {
		t.Fatalf("snapshot=%+v want Addr=reloaded Level=7", snap)
	}
	// Startup snapshot stays immutable; readers must not see torn state.
	if startup.Addr != "" || startup.Level != 0 {
		t.Fatalf("startup snapshot mutated: %+v", startup)
	}
}

func TestConf_ConcurrentReadDuringReload(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	a := NewConfig[reloadTestConfig]()
	a.Start()
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					c := a.Conf()
					// Snapshot must be consistent: Addr and Level publish together.
					if c.Addr != "" && c.Addr != fmt.Sprintf("h%d", c.Level) {
						panic(fmt.Sprintf("torn snapshot: %+v", c))
					}
				}
			}
		}()
	}
	for i := range 50 {
		if err := a.Viper.MergeConfigMap(map[string]any{"addr": fmt.Sprintf("h%d", i), "level": i}); err != nil {
			t.Fatal(err)
		}
		a.reloadConfig()
	}
	close(stop)
	wg.Wait()
	c := a.Conf()
	if c.Addr != "h49" || c.Level != 49 {
		t.Fatalf("final snapshot=%+v want h49/49", c)
	}
}

func TestLocalWatch_HotReloadSwapsSnapshot(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })

	dir := t.TempDir()
	confPath := filepath.Join(dir, "config.toml")
	localPath := filepath.Join(dir, "local.toml")
	// TOML string must use forward slashes; Windows backslashes look like escapes.
	conf := "[dev]\n[dev.localconfig]\nWatch = true\nPaths = [\"" + filepath.ToSlash(localPath) + "\"]\n"
	if err := os.WriteFile(confPath, []byte(conf), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localPath, []byte("addr = \"first\"\nlevel = 1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	os.Args = []string{"test", "-c", confPath}

	a := NewConfig[reloadTestConfig]()
	t.Cleanup(a.Start())

	startup := a.Conf()
	if startup.Addr != "first" || startup.Level != 1 {
		t.Fatalf("initial snapshot=%+v want first/1", startup)
	}

	// fsnotify debounce is 1s; rewrite the file until the snapshot swaps.
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if err := os.WriteFile(localPath, []byte("addr = \"second\"\nlevel = 2\n"), 0644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(600 * time.Millisecond)
		if snap := a.Conf(); snap.Addr == "second" {
			if snap.Level != 2 {
				t.Fatalf("torn snapshot after reload: %+v", snap)
			}
			if snap == startup {
				t.Fatal("hot reload should publish a new snapshot instance")
			}
			if startup.Addr != "first" || startup.Level != 1 {
				t.Fatalf("startup snapshot mutated by hot reload: %+v", startup)
			}
			return
		}
	}
	t.Fatalf("hot reload not observed, snapshot=%+v", a.Conf())
}

func TestNewGlobal_PublishesSnapshotAndCleansUp(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	a := New[UserConfig, EmbeddedPresets]()
	if a.Dao == nil {
		t.Fatal("Dao should be allocated by New")
	}
	// New is a pure constructor: no snapshot before Start.
	if a.Conf() != nil {
		t.Fatal("New must not publish a snapshot; Start does")
	}
	a.Start()()
	if a.Conf() == nil {
		t.Fatal("snapshot should be published after Start")
	}
}

// writeTempConfig writes content to a throwaway config.toml and returns its path.
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	confPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(confPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return confPath
}

// Tag defaults sit at the bottom of the chain: config file beats them, and the
// command line beats the config file. Env from the file must beat default:dev.
func TestConfigFileBeatsTagDefault(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test", "-c", writeTempConfig(t,
		"Name = \"hoper\"\nEnv = \"local\"\n\n[local]\nDebug = false\n")}

	a := NewConfig[reloadTestConfig]()
	t.Cleanup(a.Start())

	if a.RootConfig.Env != "local" {
		t.Fatalf("Env=%q want local, config file must beat tag default:dev", a.RootConfig.Env)
	}
	if a.RootConfig.Debug {
		t.Fatal("Debug=true want false from [local] section")
	}
}

func TestCommandLineBeatsConfigFile(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test", "-c", writeTempConfig(t,
		"Name = \"hoper\"\nEnv = \"local\"\n\n[local]\nDebug = false\n"), "-e", "prod"}

	a := NewConfig[reloadTestConfig]()
	t.Cleanup(a.Start())

	if a.RootConfig.Env != "prod" {
		t.Fatalf("Env=%q want prod, command line must beat config file", a.RootConfig.Env)
	}
}

func TestApplyTagDefaultsFillsAllKinds(t *testing.T) {
	type sample struct {
		Env   string `init:"flag:env;default:dev"`
		Port  int    `init:"flag:port;default:8080"`
		Debug bool   `init:"flag:debug;default:true"`
	}
	c := &sample{}
	applyTagDefaults(c)
	if c.Env != "dev" {
		t.Fatalf("Env=%q want dev", c.Env)
	}
	if c.Port != 8080 {
		t.Fatalf("Port=%d want 8080", c.Port)
	}
	if !c.Debug {
		t.Fatal("Debug=false want true from tag default")
	}
}

func TestTagDefaultAppliesWhenConfigOmitsEnv(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test", "-c", writeTempConfig(t, "Name = \"hoper\"\n")}

	a := NewConfig[reloadTestConfig]()
	t.Cleanup(a.Start())

	if a.RootConfig.Env != "dev" {
		t.Fatalf("Env=%q want dev from tag default when config omits Env", a.RootConfig.Env)
	}
}
