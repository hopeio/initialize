/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package http

import (
	http_fs "github.com/hopeio/utils/net/http/fs"
	http_fs_watch "github.com/hopeio/utils/net/http/fs/watch"
	"io"
	"net/http"
	"time"
)

var ConfigCenter = &Http{}

type Http struct {
	Interval  time.Duration
	Url       string
	AuthBasic string
	Headers   map[string]string
}

func (cc *Http) Type() string {
	return "http"
}

// 本地配置
func (cc *Http) Handle(handle func(io.Reader)) error {

	if cc.Interval == 0 {
		file, err := http_fs.FetchFile(cc.Url, func(r *http.Request) {
			if cc.AuthBasic != "" {
				r.Header.Add("Authorization", cc.AuthBasic)
			}
			if cc.Headers != nil {
				for k, v := range cc.Headers {
					r.Header.Add(k, v)
				}
			}
		})
		if err != nil {
			return err
		}
		handle(file.Body)
		return file.Body.Close()
	}

	watch := http_fs_watch.New(time.Second * cc.Interval)

	callback := func(hfile *http_fs.FileInfo) {
		handle(hfile.Body)
		hfile.Body.Close()
	}

	return watch.Add(cc.Url, callback)
}
