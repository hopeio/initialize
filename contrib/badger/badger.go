/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package badger

import (
	"github.com/dgraph-io/badger/v4"
)

type Config badger.Options

// BeforeInject is a no-op satisfying the Config interface.
func (c *Config) BeforeInject() {
}

// AfterInject is a no-op satisfying the Config interface.
func (c *Config) AfterInject() {

}

// Build opens a Badger database with the configured options.
func (c *Config) Build() (*badger.DB, error) {
	return badger.Open(badger.Options(*c))
}

type DB struct {
	*badger.DB
	Conf Config
}

// Config returns the embedded Conf as the configuration for injection.
func (c *DB) Config() any {
	return &c.Conf
}

// Init opens the Badger database and stores it.
func (c *DB) Init() error {
	var err error
	c.DB, err = c.Conf.Build()
	return err
}

// Close closes the Badger database if it was initialized.
func (c *DB) Close() error {
	if c.DB == nil {
		return nil
	}
	return c.DB.Close()
}
