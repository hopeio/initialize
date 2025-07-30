/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package elasticsearch

import (
	"github.com/elastic/go-elasticsearch/v8"
	http2 "github.com/hopeio/gox/net/http"
	"net/http"
)

type Config elasticsearch.Config

func (c *Config) BeforeInject() {
}
func (c *Config) AfterInject() {
	if c.Header == nil {
		c.Header = http.Header{}
	}
}

func (c *Config) Build() (*elasticsearch.Client, error) {
	if c.Username != "" && c.Password != "" {
		http2.SetBasicAuth(c.Header, c.Username, c.Password)
	}
	return elasticsearch.NewClient((elasticsearch.Config)(*c))
}

type Client struct {
	*elasticsearch.Client
	Conf Config
}

func (es *Client) Config() any {
	return &es.Conf
}

func (es *Client) Init() error {
	var err error
	es.Client, err = es.Conf.Build()
	return err
}

func (es *Client) Close() error {
	return nil
}
