/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package elastic

import (
	"github.com/olivere/elastic/v7"
	"github.com/olivere/elastic/v7/config"
)

type Config config.Config

func (c *Config) BeforeInject() {
}

func (c *Config) AfterInject() {

}

func (c *Config) Build() (*elastic.Client, error) {
	return elastic.NewClientFromConfig((*config.Config)(c))
}

type Client struct {
	*elastic.Client
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
