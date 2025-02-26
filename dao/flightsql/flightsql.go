/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package flightsql

import (
	"database/sql"
	_ "github.com/apache/arrow-adbc/go/adbc/sqldriver/flightsql"
)

type Config struct {
	DNS string
}

func (c *Config) BeforeInject() {

}

func (c *Config) AfterInject() {
}
func (c *Config) Build() (*sql.DB, error) {
	return sql.Open("flightsql", c.DNS)
}

type DB struct {
	*sql.DB
	Conf Config
}

func (m *DB) Config() any {
	return &m.Conf
}

func (m *DB) Init() error {
	var err error
	m.DB, err = m.Conf.Build()
	return err
}

func (m *DB) Close() error {
	return m.DB.Close()
}
