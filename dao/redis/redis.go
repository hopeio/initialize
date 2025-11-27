/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package redis

import (
	"context"
	"github.com/hopeio/gox/crypto/tls"
	"github.com/hopeio/gox/log"
	"time"

	"github.com/go-redis/redis/v8"
)

type Config struct {
	redis.Options
	CertFile string `json:"cert_file,omitempty"`
	KeyFile  string `json:"key_file,omitempty"`
}

func (c *Config) BeforeInject() {
}

func (c *Config) AfterInject() {
	tlsConfig, err := tls.NewServerTLSConfig(c.CertFile, c.KeyFile)
	if err != nil {
		log.Fatal(err)
	}
	c.TLSConfig = tlsConfig
	log.ValueLevelNotify("IdleTimeout", c.IdleTimeout, time.Second)
}

func (c *Config) Build() (*redis.Client, error) {
	client := redis.NewClient(&c.Options)
	return client, client.Ping(context.Background()).Err()
}

type Client struct {
	*redis.Client
	Conf Config
}

func (db *Client) Config() any {
	return &db.Conf
}

func (db *Client) Init() error {
	var err error
	db.Client, err = db.Conf.Build()
	return err
}

func (db *Client) Close() error {
	if db.Client == nil {
		return nil
	}
	return db.Client.Close()
}
