/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hopeio/gox/log"
	stringsx "github.com/hopeio/gox/text/encoding/ascii"
)

type ConfigType string

type ConfigCenter interface {
	Config() any
	io.Closer
	Handle(ctx context.Context, merge func(io.Reader) error, onChange func(io.Reader) error) error
	Type() string
}

type ConfigCenterConfig struct {
	// 配置格式
	Format string `flag:"name:format;usage:配置格式"`
	// 配置类型
	Type string `flag:"name:conf_type;usage:配置类型"`
	// config字段顺序不能变,ConfigCenter 保持在最后
	ConfigCenter ConfigCenter
}

var (
	configCenterMu sync.RWMutex
	configCenter   = map[string]ConfigCenter{}
)

// RegisterConfigCenter registers a ConfigCenter implementation by its type name (lowercase letters only).
// Duplicate registrations are silently ignored.
func RegisterConfigCenter(c ConfigCenter) {
	if c == nil {
		return
	}
	typ := strings.ToLower(c.Type())
	if !stringsx.IsAllLetter(typ) {
		log.Fatal("config type must be letters")
	}
	configCenterMu.Lock()
	defer configCenterMu.Unlock()
	if _, ok := configCenter[typ]; !ok {
		configCenter[typ] = c
	}
}

// GetConfigCenter returns the registered ConfigCenter for the given type string, or nil if not found.
func GetConfigCenter(configType string) ConfigCenter {
	configCenterMu.RLock()
	defer configCenterMu.RUnlock()
	return configCenter[configType]
}

// GetRegisteredConfigCenter returns a snapshot copy of all currently registered ConfigCenter instances.
func GetRegisteredConfigCenter() map[string]ConfigCenter {
	configCenterMu.RLock()
	defer configCenterMu.RUnlock()
	cp := make(map[string]ConfigCenter, len(configCenter))
	for k, v := range configCenter {
		cp[k] = v
	}
	return cp
}

type Client interface {
	Get() ([]byte, error)
	Set(func([]byte)) error
	Listener(func([]byte)) error
}


type Local struct {
	Watch   bool
	Paths   []string
	watcher *fsnotify.Watcher
	modTime []time.Time
}

// Type returns the identifier string "local" for this config source.
func (ld *Local) Type() string {
	return "local"
}

// Config returns the Local struct itself as its own configuration.
func (ld *Local) Config() any {
	return ld
}

// Close stops the fsnotify watcher, if one is running.
func (ld *Local) Close() error {
	if ld.watcher != nil {
		return ld.watcher.Close()
	}
	return nil
}

// Load will unmarshal configurations to struct from files that you provide
func (ld *Local) Handle(ctx context.Context, merge func(io.Reader) error, onChange func(io.Reader) error) (err error) {
	if len(ld.Paths) == 0 {
		return errors.New("empty local config path")
	}
	now := time.Now()
	ld.modTime = make([]time.Time, len(ld.Paths))
	for i, path := range ld.Paths {
		if err = load(merge, path); err != nil {
			return err
		}
		ld.modTime[i] = now
	}

	if ld.Watch {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return err
		}
		// 部分 Add 失败时关闭已创建的 watcher，避免 fd 泄漏
		defer func() {
			if err != nil {
				watcher.Close()
			}
		}()
		for _, path := range ld.Paths {
			if err = watcher.Add(path); err != nil {
				return err
			}
		}
		ld.watcher = watcher
		go ld.watchNotify(onChange)
	}

	return nil
}

// watchNotify is the goroutine that listens for fsnotify write events and calls onChange,
// debouncing rapid writes with a 1-second minimum interval per path.
func (ld *Local) watchNotify(onChange func(reader io.Reader) error) {
	// 路径可能重复，预先建立 path->index 映射，避免 slices.Index 定位错误
	pathIndex := make(map[string]int, len(ld.Paths))
	for i, p := range ld.Paths {
		pathIndex[p] = i
	}
	for {
		select {
		case event, ok := <-ld.watcher.Events:
			if !ok {
				return
			}
			log.Debugf("watch event: %v", event)
			if event.Op&fsnotify.Write == fsnotify.Write {
				idx, ok := pathIndex[event.Name]
				if !ok {
					continue
				}
				now := time.Now()
				if now.Sub(ld.modTime[idx]) < time.Second {
					continue
				}
				ld.modTime[idx] = now
				if err := load(onChange, ld.Paths[idx]); err != nil {
					log.Errorf("failed to reload data from %v, got error %v", ld.Paths[idx], err)
				}
			}
		case err, ok := <-ld.watcher.Errors:
			if !ok {
				return
			}
			log.Error(err)
		}
	}
}

// load opens a file and passes it to handle (merge or onChange callback).
func load(handle func(io.Reader) error, filepath string) (err error) {
	log.Infof("load config from: '%v'", filepath)
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	return handle(file)
}
