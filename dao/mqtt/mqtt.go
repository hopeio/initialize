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

func (c *Config) BeforeInject() {
	c.ClientOptions = mqtt.NewClientOptions()
}

func (c *Config) AfterInject() {
	c.Init()
}

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

func (c *Client) Config() any {
	return &c.Conf
}

func (c *Client) Init() error {
	var err error
	c.Client, err = c.Conf.Build()
	return err
}

func (c *Client) Close() error {
	c.Client.Disconnect(5)
	return nil
}
