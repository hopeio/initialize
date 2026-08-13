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

func TestGlobalConfig_Cleanup_RunsDefers(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	gc := NewGlobalConfig[UserConfig]()
	var deferCalled bool
	gc.Defer(func() { deferCalled = true })
	gc.Cleanup()
	if !deferCalled {
		t.Fatal("Cleanup should invoke Defer callbacks")
	}
}

func TestGlobalConfig_Cleanup_DefersReverseOrder(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	gc := NewGlobalConfig[UserConfig]()
	var order []int
	gc.Defer(func() { order = append(order, 1) })
	gc.Defer(func() { order = append(order, 2) })
	gc.Cleanup()
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
	// 普通字段/未导出字段/nil 指针曾导致 closeDao panic
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

// reloadTestConfig 字段刻意不带 flag tag，也避开 HOST/PORT 等常见环境变量名
type reloadTestConfig struct {
	EmbeddedPresets
	Addr  string
	Level int
}

func TestReloadConfig_SnapshotSwap(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	gc := NewGlobalConfig[reloadTestConfig]()
	startup := gc.Conf()
	if startup == nil {
		t.Fatal("initial snapshot should be published after NewGlobalConfig returns")
	}

	if err := gc.Viper.MergeConfigMap(map[string]any{"addr": "reloaded", "level": 7}); err != nil {
		t.Fatal(err)
	}
	gc.reloadConfig()

	snap := gc.Conf()
	if snap == startup {
		t.Fatal("reload should publish a new snapshot instance")
	}
	if snap.Addr != "reloaded" || snap.Level != 7 {
		t.Fatalf("snapshot=%+v want Addr=reloaded Level=7", snap)
	}
	// 启动快照保持不可变，读方不会看到撕裂状态
	if startup.Addr != "" || startup.Level != 0 {
		t.Fatalf("startup snapshot mutated: %+v", startup)
	}
}

func TestConf_ConcurrentReadDuringReload(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	gc := NewGlobalConfig[reloadTestConfig]()
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
					c := gc.Conf()
					// 快照内部必须一致：Addr 与 Level 同批发布
					if c.Addr != "" && c.Addr != fmt.Sprintf("h%d", c.Level) {
						panic(fmt.Sprintf("torn snapshot: %+v", c))
					}
				}
			}
		}()
	}
	for i := range 50 {
		if err := gc.Viper.MergeConfigMap(map[string]any{"addr": fmt.Sprintf("h%d", i), "level": i}); err != nil {
			t.Fatal(err)
		}
		gc.reloadConfig()
	}
	close(stop)
	wg.Wait()
	c := gc.Conf()
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
	conf := "[dev]\n[dev.localconfig]\nWatch = true\nPaths = [\"" + localPath + "\"]\n"
	if err := os.WriteFile(confPath, []byte(conf), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localPath, []byte("addr = \"first\"\nlevel = 1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	os.Args = []string{"test", "-c", confPath}

	gc := NewGlobalConfig[reloadTestConfig]()
	t.Cleanup(gc.Cleanup)

	startup := gc.Conf()
	if startup.Addr != "first" || startup.Level != 1 {
		t.Fatalf("initial snapshot=%+v want first/1", startup)
	}

	// fsnotify 防抖 1 秒，轮询期间周期性重写文件直到快照被替换
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if err := os.WriteFile(localPath, []byte("addr = \"second\"\nlevel = 2\n"), 0644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(600 * time.Millisecond)
		if snap := gc.Conf(); snap.Addr == "second" {
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
	t.Fatalf("hot reload not observed, snapshot=%+v", gc.Conf())
}

func TestNewGlobal_PublishesSnapshotAndCleansUp(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	gc := NewGlobal[UserConfig, EmbeddedPresets]()
	if gc.Conf() == nil {
		t.Fatal("snapshot should be published after NewGlobal returns")
	}
	if gc.Dao == nil {
		t.Fatal("Dao should be allocated by NewGlobal")
	}
	gc.Cleanup()
}
