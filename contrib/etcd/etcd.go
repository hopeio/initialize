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

// BeforeInject is a no-op satisfying the Config interface.
func (c *Config) BeforeInject() {
}

// AfterInject is a no-op satisfying the Config interface.
func (c *Config) AfterInject() {
}

// Build creates an etcd v3 client with the given configuration.
func (c *Config) Build() (*clientv3.Client, error) {
	return clientv3.New((clientv3.Config)(*c))
}

type Client struct {
	*clientv3.Client
	Conf Config
}

// Config returns the embedded Conf as the configuration for injection.
func (e *Client) Config() any {
	return &e.Conf
}

// Init creates the etcd client and stores it in Client.
func (e *Client) Init() error {
	var err error
	e.Client, err = e.Conf.Build()
	return err
}

// Close closes the etcd client connection.
func (e *Client) Close() error {
	if e.Client == nil {
		return nil
	}
	return e.Client.Close()
}
