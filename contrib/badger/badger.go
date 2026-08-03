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

func (c *Config) BeforeInject() {
}
func (c *Config) AfterInject() {

}
func (c *Config) Build() (*badger.DB, error) {
	return badger.Open(badger.Options(*c))
}

type DB struct {
	*badger.DB
	Conf Config
}

func (c *DB) Config() any {
	return &c.Conf
}

func (c *DB) Init() error {
	var err error
	c.DB, err = c.Conf.Build()
	return err
}

func (c *DB) Close() error {
	return c.DB.Close()
}
