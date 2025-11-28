/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package gormdb

import (
	stdlog "log"
	"os"
	"time"

	dbi "github.com/hopeio/gox/database/sql"
	loggeri "github.com/hopeio/gox/database/sql/gorm/logger"
	"github.com/hopeio/gox/log"
	"github.com/hopeio/initialize/rootconf"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/plugin/prometheus"
)

type Config struct {
	Type, Charset, Database, TimeZone string
	Host                              string
	Port                              int32
	User, Password                    string
	Postgres                          PostgresConfig
	Mysql                             MysqlConfig
	Sqlite                            SqliteConfig
	MaxIdleConns, MaxOpenConns        int
	ConnMaxLifetime, ConnMaxIdleTime  time.Duration

	Gorm gorm.Config

	UseGormLogger bool
	Logger        logger.Config

	NamingStrategy schema.NamingStrategy

	Prometheus PrometheusConfig
}

type PrometheusConfig struct {
	Enabled bool
	prometheus.Config
	MetricsCollectors []MetricsCollectorConfig
}

type PostgresConfig struct {
	Schema  string
	SSLMode string
}

type MysqlConfig struct {
	ParseTime string
	Loc       string
}

type SqliteConfig struct {
	DSN string
}

type MetricsCollectorConfig struct {
	Prefix        string
	Interval      uint32
	VariableNames []string
}

func (c *Config) Init() {
	if c.Type == "" {
		c.Type = dbi.Postgres
	}
	log.ValueLevelNotify("SlowThreshold", c.Logger.SlowThreshold, 10*time.Millisecond)
	if c.TimeZone == "" {
		c.TimeZone = "Asia/Shanghai"
	}
	if c.Postgres.SSLMode == "" {
		c.Postgres.SSLMode = "disable"
	}
	if c.Mysql.Loc == "" {
		c.Mysql.Loc = "Local"
	}
	if c.Mysql.ParseTime == "" {
		c.Mysql.ParseTime = "True"
	}
	if c.Charset == "" {
		if c.Type == dbi.Mysql {
			c.Charset = "utf8mb4"
		}
		if c.Type == dbi.Postgres {
			c.Charset = "utf8"
		}

	}

	if c.Port == 0 {
		if c.Type == dbi.Mysql {
			c.Port = 3306
		}
		if c.Type == dbi.Postgres {
			c.Port = 5432
		}
	}

	if c.Sqlite.DSN == "" {
		c.Sqlite.DSN = "./sqlite.db"
	}
}

func (c *Config) BeforeInjectWithRoot(conf *rootconf.RootConfig) {
	c.UseGormLogger = conf.Debug
	if c.Charset == "" {
		c.Charset = "utf8mb4"
	}
}

func (c *Config) AfterInject() {
	c.Init()
}

func (c *Config) Build(dialector gorm.Dialector) (*gorm.DB, error) {

	dbConfig := &c.Gorm
	dbConfig.NamingStrategy = c.NamingStrategy

	// 日志
	if c.UseGormLogger {
		// 默认日志
		logger.Default = logger.New(stdlog.New(os.Stdout, "\r", stdlog.LstdFlags), c.Logger)
	} else {
		logger.Default = &loggeri.Logger{Logger: log.NoCallerLogger().Logger, Config: &c.Logger}
	}

	db, err := gorm.Open(dialector, dbConfig)
	if err != nil {
		return nil, err
	}

	if c.Prometheus.Enabled {
		if c.Type == dbi.Mysql {
			for _, pc := range c.Prometheus.MetricsCollectors {
				c.Prometheus.MetricsCollector = append(c.Prometheus.MetricsCollector, &prometheus.MySQL{
					Prefix:        pc.Prefix,
					Interval:      pc.Interval,
					VariableNames: pc.VariableNames,
				})
			}

		}
		if c.Type == dbi.Postgres {
			for _, pc := range c.Prometheus.MetricsCollectors {
				c.Prometheus.MetricsCollector = append(c.Prometheus.MetricsCollector, &prometheus.Postgres{
					Prefix:        pc.Prefix,
					Interval:      pc.Interval,
					VariableNames: pc.VariableNames,
				})
			}
		}
		err = db.Use(prometheus.New(c.Prometheus.Config))
		if err != nil {
			return nil, err
		}
	}

	rawDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	rawDB.SetMaxIdleConns(c.MaxIdleConns)
	rawDB.SetMaxOpenConns(c.MaxOpenConns)
	rawDB.SetConnMaxLifetime(c.ConnMaxLifetime)
	rawDB.SetConnMaxIdleTime(c.ConnMaxIdleTime)
	return db, nil
}

type DB struct {
	*gorm.DB
	Conf Config
}

func (db *DB) Table(name string) *gorm.DB {
	gdb := db.DB.Clauses()
	gdb.Statement.TableExpr = &clause.Expr{SQL: gdb.Statement.Quote(name)}
	gdb.Statement.Table = name
	return gdb
}
