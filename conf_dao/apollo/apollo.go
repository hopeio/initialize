/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package apollo

import (
	"github.com/hopeio/utils/dao/apollo"
)

type Config apollo.Config

func (c *Config) BeforeInject() {
}

func (c *Config) Init() {

}

func (c *Config) Build() (*apollo.Client, error) {
	c.Init()
	//初始化更新配置，这里不需要，开启实时更新时初始化会更新一次
	return (*apollo.Config)(c).NewClient()
}

type Client struct {
	*apollo.Client
	Conf Config
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
	return c.Client.Close()
}
