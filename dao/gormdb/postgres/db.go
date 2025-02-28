/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package postgres

import (
	"fmt"
	pkdb "github.com/hopeio/initialize/dao/gormdb"
	"github.com/hopeio/initialize/rootconf"
	dbi "github.com/hopeio/utils/dao/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Config pkdb.Config

func (c *Config) BeforeInjectWithRoot(conf *rootconf.RootConfig) {
	(*pkdb.Config)(c).BeforeInjectWithRoot(conf)
}

func (c *Config) AfterInject() {
	(*pkdb.Config)(c).AfterInject()
}

func (c *Config) Build() (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s dbname=%s port=%d sslmode=%s password=%s TimeZone=%s",
		c.Host, c.User, c.Database, c.Port, c.Postgres.SSLMode, c.Password, c.TimeZone)
	return (*pkdb.Config)(c).Build(postgres.Open(dsn))
}

type DB pkdb.DB

func (db *DB) Config() any {
	return (*Config)(&db.Conf)
}

func (db *DB) Init() error {
	var err error
	db.Conf.Type = dbi.Mysql
	db.DB, err = (*Config)(&db.Conf).Build()
	return err
}

func (db *DB) Close() error {
	return nil
}

func (db *DB) Table(name string) *gorm.DB {
	name = db.TableName(name)
	gdb := db.DB.Clauses()
	gdb.Statement.TableExpr = &clause.Expr{SQL: gdb.Statement.Quote(name)}
	gdb.Statement.Table = name
	return gdb
}

func (db *DB) TableName(name string) string {
	if db.Conf.Postgres.Schema != "" {
		return db.Conf.Postgres.Schema + "." + name
	}
	return name
}

// TODO:
func (db *DB) Inject(configName string) {

}
