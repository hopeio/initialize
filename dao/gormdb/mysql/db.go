/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package mysql

import (
	"fmt"

	dbi "github.com/hopeio/gox/database/sql"
	pkdb "github.com/hopeio/initialize/dao/gormdb"
	"github.com/hopeio/initialize/rootconf"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Config pkdb.Config

func (c *Config) BeforeInjectWithRoot(conf *rootconf.RootConfig) {
	(*pkdb.Config)(c).BeforeInjectWithRoot(conf)
}
func (c *Config) AfterInject() {
	(*pkdb.Config)(c).AfterInject()
}

func (c *Config) Build() (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%s&loc=%s",
		c.User, c.Password, c.Host,
		c.Port, c.Database, c.Charset, c.Mysql.ParseTime, c.Mysql.Loc)
	return (*pkdb.Config)(c).Build(mysql.Open(dsn))
}

type DB pkdb.DB

func (db *DB) Config() any {
	return (*Config)(&db.Conf)
}

func (db *DB) Init() error {
	var err error
	db.Conf.Type = dbi.Postgres
	db.DB, err = (*Config)(&db.Conf).Build()
	return err
}

func (db *DB) Close() error {
	return nil
}
