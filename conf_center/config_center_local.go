/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package conf_center

import (
	"errors"
	"io"
	"os"
	"slices"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hopeio/gox/log"
)

type Local struct {
	// 间隔大于1秒采用timer定时加载，小于1秒用fsnotify
	Watch          bool
	WatchReload    bool
	ReloadInterval time.Duration
	Paths          []string
	watcher        *fsnotify.Watcher
	timer          *time.Ticker
	modTime        []time.Time
}

func (ld *Local) Type() string {
	return "local"
}

func (ld *Local) Config() any {
	return ld
}

func (ld *Local) Close() error {
	if ld.watcher != nil {
		return ld.watcher.Close()
	}
	if ld.timer != nil {
		ld.timer.Stop()
	}
	return nil
}

// Load will unmarshal configurations to struct from files that you provide
func (ld *Local) Handle(handle func(io.Reader) error, done func() error) (err error) {
	if len(ld.Paths) == 0 {
		return errors.New("empty local config path")
	}
	now := time.Now()
	ld.modTime = make([]time.Time, len(ld.Paths))
	for i, path := range ld.Paths {
		err = load(handle, path)
		if err != nil {
			return err
		}
		ld.modTime[i] = now
	}

	if ld.ReloadInterval == 0 {
		ld.ReloadInterval = time.Second
	}

	if ld.WatchReload {
		ld.timer = time.NewTicker(ld.ReloadInterval)
		go ld.watchTimer(handle, done)
	} else {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			ld.timer = time.NewTicker(ld.ReloadInterval)
			go ld.watchTimer(handle, done)
			return nil
		} else {
			for _, path := range ld.Paths {
				err = watcher.Add(path)
				if err != nil {
					return err
				}
			}
			ld.watcher = watcher
			go ld.watchNotify(handle, done)
		}
	}

	return
}

func (ld *Local) watchNotify(handle func(reader io.Reader) error, done func() error) {
	for {
		select {
		case event, ok := <-ld.watcher.Events:
			if !ok {
				return
			}
			log.Debugf("watch event: %v", event)
			if event.Op&fsnotify.Write == fsnotify.Write {
				idx := slices.Index(ld.Paths, event.Name)
				now := time.Now()
				if now.Sub(ld.modTime[idx]) < time.Second {
					continue
				}
				ld.modTime[idx] = now
				if err := load(handle, ld.Paths[idx]); err != nil {
					log.Errorf("failed to reload data from %v, got error %v\n", ld.Paths, err)
				} else {
					done()
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

func (ld *Local) watchTimer(handle func(reader io.Reader) error, done func() error) {

	for range ld.timer.C {
		for i := range ld.Paths {
			// check configuration
			if fileInfo, err := os.Stat(ld.Paths[i]); err == nil && fileInfo.Mode().IsRegular() {
				if fileInfo.ModTime().After(ld.modTime[i]) {
					ld.modTime[i] = fileInfo.ModTime()
					if err := load(handle, ld.Paths[i]); err != nil {
						log.Error("failed to reload data from %v, got error %v\n", ld.Paths, err)
					} else {
						done()
					}
					break
				}
			}
		}
	}
}

func load(handle func(io.Reader) error, filepath string) (err error) {
	log.Debugf("load config from: '%v'", filepath)
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	err = handle(file)
	if err != nil {
		return err
	}
	return file.Close()
}
