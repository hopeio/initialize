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

	httpx "github.com/hopeio/gox/net/http"
)

var ConfigCenter = &Http{}

type Http struct {
	ReloadInterval time.Duration
	Urls           []string
	AuthBasic      string
	Headers        map[string]string
	modTime        []time.Time
}

// Type returns the identifier string "http" for this config center.
func (cc *Http) Type() string {
	return "http"
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
		merge(file.Body)
		err = file.Body.Close()
		if err != nil {
			return err
		}
	}

	if cc.ReloadInterval > 0 {
		watch := httpx.NewFileWatcher(time.Second * cc.ReloadInterval)

		callback := func(hfile *httpx.FileInfo) {
			onChange(hfile.Body)
			hfile.Body.Close()
		}

		for _, url := range cc.Urls {
			err := watch.Add(url, callback)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
