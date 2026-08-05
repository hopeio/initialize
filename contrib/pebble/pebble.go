/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package pebble

import (
	"errors"

	"github.com/cockroachdb/pebble/v2"
)

type Config struct {
	DirName string
	pebble.Options
}

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

// Build opens a Pebble database at the configured directory.
func (c *Config) Build() (*pebble.DB, error) {
	if c.DirName == "" {
		return nil, errors.New("pebble dir name is empty")
	}
	return pebble.Open(c.DirName, &c.Options)
}

type DB struct {
	*pebble.DB
	Conf Config
}

// Config returns the embedded Conf as the configuration for injection.
func (p *DB) Config() any {
	return &p.Conf
}

// Init opens the Pebble database and stores it.
func (p *DB) Init() error {
	var err error
	p.DB, err = p.Conf.Build()
	return err
}

// Close closes the Pebble database.
func (p *DB) Close() error {
	if p.DB == nil {
		return nil
	}
	return p.DB.Close()
}
