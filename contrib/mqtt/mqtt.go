/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package mqtt

import (
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hopeio/gox/crypto/tls"
	"github.com/hopeio/gox/log"
)

type Config struct {
	*mqtt.ClientOptions
	Brokers    []string
	CAFile     string `json:"ca_file,omitempty"`
	ServerName string
}

// BeforeInject initializes the embedded ClientOptions with MQTT defaults.
func (c *Config) BeforeInject() {
	c.ClientOptions = mqtt.NewClientOptions()
}

// AfterInject calls Init to finalize broker addresses and TLS config.
func (c *Config) AfterInject() {
	c.Init()
}

// Init sets up TLS and broker addresses, logs warnings for unset timeouts.
func (c *Config) Init() *Config {
	if c.CAFile != "" && c.ServerName != "" {
		tlsConfig, err := tls.NewClientTLSConfig(c.CAFile, c.ServerName)
		if err != nil {
			log.Fatal(err)
		}
		c.TLSConfig = tlsConfig
	}

	for _, broker := range c.Brokers {
		c.ClientOptions.AddBroker(broker)
	}

	log.ValueLevelNotify("PingTimeout", c.PingTimeout, time.Second)
	log.ValueLevelNotify("ConnectTimeout", c.ConnectTimeout, time.Second)
	log.ValueLevelNotify("MaxReconnectInterval", c.MaxReconnectInterval, time.Second)
	log.ValueLevelNotify("ConnectRetryInterval", c.ConnectRetryInterval, time.Second)
	log.ValueLevelNotify("WriteTimeout", c.WriteTimeout, time.Second)
	return c
}

// Build creates an MQTT client, connects to all configured brokers, and returns it.
func (c *Config) Build() (mqtt.Client, error) {
	client := mqtt.NewClient(c.ClientOptions)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return client, token.Error()
	}
	return client, nil
}

type Client struct {
	Conf Config
	mqtt.Client
}

// Config returns the embedded Conf as the configuration for injection.
func (c *Client) Config() any {
	return &c.Conf
}

// Init creates and connects the MQTT client.
func (c *Client) Init() error {
	var err error
	c.Client, err = c.Conf.Build()
	return err
}

// Close disconnects the MQTT client with a 5ms quiesce period.
func (c *Client) Close() error {
	if c.Client != nil {
		c.Client.Disconnect(5)
	}
	return nil
}
