/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package loki

import (
	"flag"
	"github.com/grafana/loki-client-go/loki"
	"github.com/grafana/loki-client-go/pkg/urlutil"
	"github.com/hopeio/gox/log"
	"os"
)

type Config struct {
	loki.Config
	Url string
}

func (c *Config) BeforeInject() {

}

func (c *Config) AfterInject() {
	c.Init()
}

func (c *Config) Init() *Config {
	return c
}

func (c *Config) Build() (*loki.Client, error) {
	var u urlutil.URLValue
	f := &flag.FlagSet{}
	c.RegisterFlags(f)
	if err := f.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
	if err := u.Set(c.Url); err != nil {
		log.Fatal(err)
	}
	c.URL = u
	return loki.New(c.Config)
}

type Client struct {
	Client *loki.Client
	Conf   Config
}

func (c *Client) Config() any {
	return &c.Conf
}

func (c *Client) Init() error {
	var err error
	c.Client, err = c.Conf.Build()
	return err
}

func (c *Client) Close() error {
	c.Client.Stop()
	return nil
}
