/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package common

import (
	"errors"

	dbi "github.com/hopeio/gox/database/sql"
	pkdb "github.com/hopeio/initialize/dao/gormdb"
	"github.com/hopeio/initialize/dao/gormdb/mysql"
	"github.com/hopeio/initialize/dao/gormdb/postgres"
	"github.com/hopeio/initialize/dao/gormdb/sqlite"
	"github.com/hopeio/initialize/rootconf"
	"gorm.io/gorm"
)

// Deprecated 每个驱动分开，不然每次都要编译所有驱动
type Config pkdb.Config

func (c *Config) BeforeInjectWithRoot(conf *rootconf.RootConfig) {
	(*pkdb.Config)(c).BeforeInjectWithRoot(conf)
}

func (c *Config) AfterInject() {
	(*pkdb.Config)(c).AfterInject()
}
func (c *Config) Build() (*gorm.DB, error) {
	if c.Type == dbi.Mysql {
		return (*mysql.Config)(c).Build()
	} else if c.Type == dbi.Postgres {
		return (*postgres.Config)(c).Build()
	} else if c.Type == dbi.Sqlite {
		return (*sqlite.Config)(c).Build()
	}

	return nil, errors.New("只支持" + dbi.Mysql + "," + dbi.Postgres + "." + dbi.Sqlite)
}

type DB pkdb.DB

func (db *DB) Config() any {
	return (*Config)(&db.Conf)
}

func (db *DB) Init() error {
	var err error
	db.DB, err = (*Config)(&db.Conf).Build()
	return err
}

func (db *DB) Close() error {
	return nil
}
