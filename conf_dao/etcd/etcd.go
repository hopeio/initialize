/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package etcd

import (
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Config clientv3.Config

func (c *Config) BeforeInject() {
}

func (c *Config) AfterInject() {
}

func (c *Config) Build() (*clientv3.Client, error) {
	return clientv3.New((clientv3.Config)(*c))
}

type Client struct {
	*clientv3.Client
	Conf Config
}

func (e *Client) Config() any {
	return &e.Conf
}

func (e *Client) Init() error {
	var err error
	e.Client, err = e.Conf.Build()
	return err
}

func (e *Client) Close() error {
	if e.Client == nil {
		return nil
	}
	return e.Client.Close()
}
