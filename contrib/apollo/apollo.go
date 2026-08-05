/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package apollo

import (
	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/env/config"
)

type Config config.AppConfig

// BeforeInject is a no-op satisfying the Config interface.
func (c *Config) BeforeInject() {
}

// AfterInject is a no-op satisfying the Config interface.
func (c *Config) AfterInject() {

}

// Build starts an Apollo client using the embedded AppConfig.
func (c *Config) Build() (agollo.Client, error) {
	//初始化更新配置，这里不需要，开启实时更新时初始化会更新一次
	return agollo.StartWithConfig(func() (*config.AppConfig, error) {
		return (*config.AppConfig)(c), nil
	})
}

type Client struct {
	agollo.Client
	Conf Config
}

// Config returns the embedded Conf as the configuration for injection.
func (c *Client) Config() any {
	return &c.Conf
}

// Init creates the Apollo client and stores it.
func (c *Client) Init() error {
	var err error
	c.Client, err = c.Conf.Build()
	return err
}

// Close stops the Apollo client.
func (c *Client) Close() error {
	c.Client.Close()
	return nil
}
