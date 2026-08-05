/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package sqlite

import (
	sqlx "github.com/hopeio/gox/database/sql"
	"github.com/hopeio/initialize"
	pkdb "github.com/hopeio/initialize/contrib/gormdb"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Config pkdb.Config

func (c *Config) BeforeInjectWithRoot(conf *initialize.RootConfig) {
	(*pkdb.Config)(c).BeforeInjectWithRoot(conf)
}
func (c *Config) AfterInject() {
	(*pkdb.Config)(c).AfterInject()
}

func (c *Config) Build() (*gorm.DB, error) {
	return (*pkdb.Config)(c).Build(sqlite.Open(c.Sqlite.DSN))
}

type DB pkdb.DB

func (db *DB) Config() any {
	return (*Config)(&db.Conf)
}

func (db *DB) Init() error {
	var err error
	db.Conf.Type = sqlx.Sqlite
	db.DB, err = (*Config)(&db.Conf).Build()
	return err
}

func (db *DB) Close() error {
	dbx, err := db.DB.DB()
	if err != nil {
		return err
	}
	return dbx.Close()
}
