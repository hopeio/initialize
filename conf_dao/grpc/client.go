/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package grpc

import (
	grpci "github.com/hopeio/utils/net/http/grpc"
	"google.golang.org/grpc"
)

type Config struct {
	Addr    string
	TLS     bool
	Options []grpc.DialOption
}

func (c *Config) BeforeInject() {
}

func (c *Config) Init() {

}

func (c *Config) Build() (*grpc.ClientConn, error) {
	c.Init()
	if c.TLS {
		return grpci.NewTLSClient(c.Addr, c.Options...)
	}
	return grpci.NewClient(c.Addr, c.Options...)

}

type Client struct {
	Conn *grpc.ClientConn
	Conf Config
}

func (c *Client) Config() any {
	return &c.Conf
}

func (c *Client) Init() error {
	var err error
	c.Conn, err = c.Conf.Build()
	return err
}

func (c *Client) Close() error {
	return c.Conn.Close()
}
