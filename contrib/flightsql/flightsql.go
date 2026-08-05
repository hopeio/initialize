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

// BeforeInject is a no-op satisfying the Config interface.
func (c *Config) BeforeInject() {

}

// AfterInject is a no-op satisfying the Config interface.
func (c *Config) AfterInject() {
}

// Build opens a FlightSQL database via ADBC.
func (c *Config) Build() (*sql.DB, error) {
	return sql.Open("flightsql", c.DNS)
}

type DB struct {
	*sql.DB
	Conf Config
}

// Config returns the embedded Conf as the configuration for injection.
func (m *DB) Config() any {
	return &m.Conf
}

// Init opens the FlightSQL database and stores it.
func (m *DB) Init() error {
	var err error
	m.DB, err = m.Conf.Build()
	return err
}

// Close closes the FlightSQL database connection.
func (m *DB) Close() error {
	return m.DB.Close()
}
