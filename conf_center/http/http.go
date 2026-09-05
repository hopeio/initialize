/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package http

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/hopeio/gox/log"
	httpx "github.com/hopeio/gox/net/http"
	"github.com/hopeio/initialize"
)

var ConfigCenter = &Http{}

var _ initialize.ConfigCenter = (*Http)(nil)

type Http struct {
	ReloadInterval time.Duration
	Urls           []string
	AuthBasic      string
	Headers        map[string]string
	watcher        *httpx.FileWatcher
}

// Type returns the identifier string "http" for this config center.
func (cc *Http) Type() string {
	return "http"
}

// Config returns the Http struct itself as its own configuration.
func (cc *Http) Config() any {
	return cc
}

// Close stops the background URL watcher, if one is running.
func (cc *Http) Close() error {
	if cc.watcher != nil {
		cc.watcher.Close()
	}
	return nil
}

// Handle fetches each URL and calls merge; if ReloadInterval > 0, starts a background watcher
// that calls onChange whenever any URL content changes.
func (cc *Http) Handle(ctx context.Context, merge func(io.Reader) error, onChange func(io.Reader) error) error {

	for _, url := range cc.Urls {
		file, err := httpx.FetchFile(url, func(r *http.Request) {
			if cc.AuthBasic != "" {
				r.Header.Add("Authorization", cc.AuthBasic)
			}
			for k, v := range cc.Headers {
				r.Header.Add(k, v)
			}
		})
		if err != nil {
			return err
		}
		mergeErr := merge(file.Body)
		if err := file.Body.Close(); err != nil {
			return err
		}
		if mergeErr != nil {
			return mergeErr
		}
	}

	if cc.ReloadInterval > 0 {
		// ReloadInterval is already a time.Duration; do not multiply by time.Second.
		watch := httpx.NewFileWatcher(cc.ReloadInterval)

		callback := func(hfile *httpx.FileInfo) {
			if err := onChange(hfile.Body); err != nil {
				log.Errorf("http config center onChange error: %v", err)
			}
			hfile.Body.Close()
		}

		for _, url := range cc.Urls {
			err := watch.Add(url, callback)
			if err != nil {
				watch.Close()
				return err
			}
		}
		cc.watcher = watch
	}
	return nil
}
