/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package nacos

import (
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type Config struct {
	vo.NacosClientParam
}

// BeforeInject is a no-op satisfying the Config interface.
func (c *Config) BeforeInject() {

}

// AfterInject calls Init after config fields are populated.
func (c *Config) AfterInject() {
	c.Init()
}

// Init is a no-op placeholder for future default initialization.
func (c *Config) Init() *Config {
	return c
}

// Build creates a Nacos config client.
func (c *Config) Build() (config_client.IConfigClient, error) {
	return clients.NewConfigClient(c.NacosClientParam)
}

type ConfigClient struct {
	Client config_client.IConfigClient
	Conf   Config
}

// Config returns the embedded Conf as the configuration for injection.
func (m *ConfigClient) Config() any {
	return &m.Conf
}

// Init creates the Nacos config client and stores it.
func (m *ConfigClient) Init() error {
	var err error
	m.Client, err = m.Conf.Build()
	return err
}

// Close closes the Nacos config client connection.
func (m *ConfigClient) Close() error {
	m.Client.CloseClient()
	return nil
}
