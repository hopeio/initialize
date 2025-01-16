/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package client

import "github.com/nats-io/nats.go"

type Config nats.Options

func (c *Config) BeforeInject() {
}
func (c *Config) AfterInject() {
	c.Init()
}

func (c *Config) Init() {
}

func (c *Config) Build() (*nats.Conn, error) {
	return (*nats.Options)(c).Connect()
}

type Client struct {
	*nats.Conn
	Conf Config
}

func (db *Client) Config() any {
	return &db.Conf
}

func (db *Client) Init() error {
	var err error
	db.Conn, err = db.Conf.Build()
	return err
}

func (db *Client) Close() error {
	if db.Conn == nil {
		return nil
	}
	db.Conn.Close()
	return nil
}
