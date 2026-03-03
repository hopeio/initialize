/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package duckdb

import (
	"database/sql"

	_ "github.com/duckdb/duckdb-go/v2"
)

// https://github.com/marcboeker/go-duckdb/issues/115
// CGO_ENABLED=1 CGO_LDFLAGS="-L/path/to/duckdb.dll" CGO_CFLAGS="-I/path/to/duckdb.h" go build -tags=duckdb_use_lib,go1.22 main.go
// unix: LD_LIBRARY_PATH=/path/to/libs ./main
// win: PATH=/path/to/libs:$PATH or copy dll to run dir or C:\Windows\System32和C:\Windows\SysWOW64 ./main

type Config struct {
	DSN string `json:"dsn"`
}

func (c *Config) BeforeInject() {

}

func (c *Config) AfterInject() {
}

func (c *Config) Build() (*sql.DB, error) {
	return sql.Open("duckdb", c.DSN)
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
