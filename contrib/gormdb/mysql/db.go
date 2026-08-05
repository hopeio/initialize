/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package mysql

import (
	"fmt"

	sqlx "github.com/hopeio/gox/database/sql"
	"github.com/hopeio/initialize"
	pkdb "github.com/hopeio/initialize/contrib/gormdb"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Config pkdb.Config

// BeforeInjectWithRoot delegates to the parent gormdb Config.
func (c *Config) BeforeInjectWithRoot(conf *initialize.RootConfig) {
	(*pkdb.Config)(c).BeforeInjectWithRoot(conf)
}

// AfterInject delegates to the parent gormdb Config.
func (c *Config) AfterInject() {
	(*pkdb.Config)(c).AfterInject()
}

// Build constructs a MySQL DSN and opens a GORM DB connection.
func (c *Config) Build() (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%s&loc=%s",
		c.User, c.Password, c.Host,
		c.Port, c.Database, c.Charset, c.Mysql.ParseTime, c.Mysql.Loc)
	return (*pkdb.Config)(c).Build(mysql.Open(dsn))
}

type DB pkdb.DB

// Config returns the embedded Conf as the configuration for injection.
func (db *DB) Config() any {
	return (*Config)(&db.Conf)
}

// Init opens the MySQL connection and stores it in DB.
func (db *DB) Init() error {
	var err error
	db.Conf.Type = sqlx.Postgres
	db.DB, err = (*Config)(&db.Conf).Build()
	return err
}

// Close closes the underlying sql.DB connection.
func (db *DB) Close() error {
	dbx, err := db.DB.DB()
	if err != nil {
		return err
	}
	return dbx.Close()
}
