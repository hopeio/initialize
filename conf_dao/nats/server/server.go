/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package server

import "github.com/nats-io/nats-server/v2/server"

type Config server.Options

func (c *Config) BeforeInject() {
}
func (c *Config) AfterInject() {
	c.Init()
}

func (c *Config) Init() {
}

func (c *Config) Build() (*server.Server, error) {
	return server.NewServer((*server.Options)(c))
}

type Server struct {
	Conf Config
	*server.Server
}

func (db *Server) Config() any {
	return &db.Conf
}

func (db *Server) Init() error {
	var err error
	db.Server, err = db.Conf.Build()
	return err
}

func (db *Server) Close() error {
	return nil
}
