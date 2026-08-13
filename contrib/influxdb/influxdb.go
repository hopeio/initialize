/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package influxdb

import influxdb2 "github.com/influxdata/influxdb-client-go/v2"

type Config struct {
	ServerURL string
	AuthToken string
}

// Build creates an InfluxDB v2 client.
func (c *Config) Build() influxdb2.Client {
	return influxdb2.NewClient(c.ServerURL, c.AuthToken)
}

type Client struct {
	Client influxdb2.Client
	Conf   Config
}

// Config returns a pointer to the embedded Conf so injection writes into it.
func (c *Client) Config() any {
	return &c.Conf
}

// Init creates the InfluxDB client and stores it.
func (c *Client) Init() error {
	c.Client = c.Conf.Build()
	return nil
}

// Close shuts down the InfluxDB client if it was initialized.
func (c *Client) Close() error {
	if c.Client != nil {
		c.Client.Close()
	}
	return nil
}
