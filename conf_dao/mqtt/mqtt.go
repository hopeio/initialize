/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package mqtt

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hopeio/utils/crypto/tls"
	"github.com/hopeio/utils/log"
	"time"
)

type Config struct {
	*mqtt.ClientOptions
	Brokers []string
	CAFile  string `json:"ca_file,omitempty"`
}

func (c *Config) BeforeInject() {
	c.ClientOptions = mqtt.NewClientOptions()
}

func (c *Config) Init() {
	tlsConfig, err := tls.NewClientTLSConfig(c.CAFile, "")
	if err != nil {
		log.Fatal(err)
	}
	c.TLSConfig = tlsConfig
	for _, broker := range c.Brokers {
		c.ClientOptions.AddBroker(broker)
	}

	log.DurationNotify("PingTimeout", c.PingTimeout, time.Second)
	log.DurationNotify("ConnectTimeout", c.ConnectTimeout, time.Second)
	log.DurationNotify("MaxReconnectInterval", c.MaxReconnectInterval, time.Second)
	log.DurationNotify("ConnectRetryInterval", c.ConnectRetryInterval, time.Second)
	log.DurationNotify("WriteTimeout", c.WriteTimeout, time.Second)
}

func (c *Config) Build() (mqtt.Client, error) {
	c.Init()
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
