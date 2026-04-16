/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package conf_center

import (
	"context"
	"errors"
	"io"
	"os"
	"slices"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hopeio/gox/log"
)

type Local struct {
	Watch   bool
	Paths   []string
	watcher *fsnotify.Watcher
	modTime []time.Time
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
	return nil
}

// Load will unmarshal configurations to struct from files that you provide
func (ld *Local) Handle(ctx context.Context, merge func(io.Reader) error, onChange func(io.Reader) error) (err error) {
	if len(ld.Paths) == 0 {
		return errors.New("empty local config path")
	}
	now := time.Now()
	for _, path := range ld.Paths {
		err = load(merge, path)
		if err != nil {
			return err
		}
		ld.modTime = append(ld.modTime, now)
	}

	if ld.Watch {
		watcher, err := fsnotify.NewWatcher()
		for _, path := range ld.Paths {
			err = watcher.Add(path)
			if err != nil {
				return err
			}
		}
		ld.watcher = watcher
		go ld.watchNotify(onChange)
	}

	return nil
}

func (ld *Local) watchNotify(onChange func(reader io.Reader) error) {
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
				if err := load(onChange, ld.Paths[idx]); err != nil {
					log.Errorf("failed to reload data from %v, got error %v", ld.Paths, err)
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

func load(handle func(io.Reader) error, filepath string) (err error) {
	log.Infof("load config from: '%v'", filepath)
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	return handle(file)
}
