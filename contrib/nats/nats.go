/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package nats

import "github.com/nats-io/nats.go"

type Config nats.Options

// BeforeInject is a no-op satisfying the Config interface.
func (c *Config) BeforeInject() {
}

// AfterInject calls Init after config fields are populated.
func (c *Config) AfterInject() {
	c.Init()
}

// Init is a no-op placeholder for future default initialization.
func (c *Config) Init() {
}

// Build connects to NATS using the stored options.
func (c *Config) Build() (*nats.Conn, error) {
	return (*nats.Options)(c).Connect()
}

type Client struct {
	*nats.Conn
	Conf Config
}

// Config returns the embedded Conf as the configuration for injection.
func (db *Client) Config() any {
	return &db.Conf
}

// Init creates the NATS connection and stores it in Conn.
func (db *Client) Init() error {
	var err error
	db.Conn, err = db.Conf.Build()
	return err
}

// Close drains the NATS connection to flush pending messages before disconnecting.
func (db *Client) Close() error {
	if db.Conn == nil {
		return nil
	}
	db.Conn.Drain()
	return nil
}
