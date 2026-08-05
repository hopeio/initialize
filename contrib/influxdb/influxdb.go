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
	options   *influxdb2.Options
}

// Build creates an InfluxDB v2 client.
func (c *Config) Build() influxdb2.Client {
	client := influxdb2.NewClientWithOptions(c.ServerURL, c.AuthToken, c.options)
	return client
}

type Client struct {
	Client influxdb2.Client
	Conf   Config
}

// Config returns the embedded Conf as the configuration for injection.
func (c *Client) Config() any {
	return c.Conf
}

// Init creates the InfluxDB client and stores it.
func (c *Client) Init() error {
	c.Client = c.Conf.Build()
	return nil
}

// Close shuts down the InfluxDB client.
func (c *Client) Close() error {
	c.Client.Close()
	return nil
}
