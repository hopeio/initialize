/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package postgres

import (
	"fmt"

	sqlx "github.com/hopeio/gox/database/sql"
	"github.com/hopeio/initialize"
	pkdb "github.com/hopeio/initialize/contrib/gormdb"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Config pkdb.Config

// BeforeInjectWithRoot sets the PostgreSQL database type before defaults are applied,
// then delegates to the parent gormdb Config.
func (c *Config) BeforeInjectWithRoot(conf *initialize.RootConfig) {
	if c.Type == "" {
		c.Type = sqlx.Postgres
	}
	(*pkdb.Config)(c).BeforeInjectWithRoot(conf)
}

// AfterInject delegates to the parent gormdb Config.
func (c *Config) AfterInject() {
	(*pkdb.Config)(c).AfterInject()
}

// Build constructs a PostgreSQL DSN and opens a GORM DB connection.
func (c *Config) Build() (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s dbname=%s port=%d sslmode=%s password=%s TimeZone=%s",
		c.Host, c.User, c.Database, c.Port, c.Postgres.SSLMode, c.Password, c.TimeZone)
	return (*pkdb.Config)(c).Build(postgres.Open(dsn))
}

type DB pkdb.DB

// Config returns the embedded Conf as the configuration for injection.
func (db *DB) Config() any {
	return (*Config)(&db.Conf)
}

// Init opens the PostgreSQL connection and stores it in DB.
func (db *DB) Init() error {
	var err error
	db.Conf.Type = sqlx.Postgres
	db.DB, err = (*Config)(&db.Conf).Build()
	return err
}

// Close closes the underlying sql.DB connection if it was initialized.
func (db *DB) Close() error {
	if db.DB == nil {
		return nil
	}
	dbx, err := db.DB.DB()
	if err != nil {
		return err
	}
	return dbx.Close()
}

// Table returns a *gorm.DB scoped to the given table name (with schema prefix if configured).
func (db *DB) Table(name string) *gorm.DB {
	name = db.TableName(name)
	gdb := db.DB.Clauses()
	gdb.Statement.TableExpr = &clause.Expr{SQL: gdb.Statement.Quote(name)}
	gdb.Statement.Table = name
	return gdb
}

// TableName returns the schema-qualified table name when a Postgres schema is configured.
func (db *DB) TableName(name string) string {
	if db.Conf.Postgres.Schema != "" {
		return db.Conf.Postgres.Schema + "." + name
	}
	return name
}
