/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package pebble

import (
	"errors"
	"github.com/cockroachdb/pebble"
)

type Config struct {
	DirName string
	pebble.Options
}

func (c *Config) BeforeInject() {
}
func (c *Config) Init() {
}

func (c *Config) Build() (*pebble.DB, error) {
	c.Init()
	if c.DirName == "" {
		return nil, errors.New("pebble dir name is empty")
	}
	return pebble.Open(c.DirName, &c.Options)
}

type DB struct {
	*pebble.DB
	Conf Config
}

func (p *DB) Config() any {
	return &p.Conf
}

func (p *DB) Init() error {
	var err error
	p.DB, err = p.Conf.Build()
	return err
}

func (p *DB) Close() error {
	if p.DB == nil {
		return nil
	}
	return p.DB.Close()
}
