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

	redisotel "github.com/redis/go-redis/extra/redisotel-native/v9"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	redis.Options
	Otel     redisotel.Config
	CertFile string `json:"cert_file,omitempty"`
	KeyFile  string `json:"key_file,omitempty"`
}

func (c *Config) BeforeInject() {
}

func (c *Config) AfterInject() {
	if c.CertFile != "" && c.KeyFile != "" {
		tlsConfig, err := tls.NewServerTLSConfig(c.CertFile, c.KeyFile)
		if err != nil {
			log.Fatal(err)
		}
		c.TLSConfig = tlsConfig
	}
}

func (c *Config) Build() (*redis.Client, error) {
	client := redis.NewClient(&c.Options)
	if c.Otel.Enabled {
		otelInstance := redisotel.GetObservabilityInstance()
		if err := otelInstance.Init(&c.Otel); err != nil {
			log.Fatalf("Failed to initialize OTel: %v", err)
		}
	}
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
	err:= db.Client.Close()
	if err != nil {
		return err
	}
	if db.Conf.Otel.Enabled {
		return redisotel.GetObservabilityInstance().Shutdown()
	}
	return nil
}
