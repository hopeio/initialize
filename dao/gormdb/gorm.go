/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package gormdb

import (
	"github.com/hopeio/initialize/rootconf"
	dbi "github.com/hopeio/utils/dao/database"
	gormi "github.com/hopeio/utils/dao/database/gorm"
	loggeri "github.com/hopeio/utils/dao/database/gorm/logger"
	"github.com/hopeio/utils/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/prometheus"
	stdlog "log"
	"os"
)

type Config gormi.Config

func (c *Config) BeforeInjectWithRoot(conf *rootconf.RootConfig) {
	c.EnableStdLogger = conf.Debug
	if c.Charset == "" {
		c.Charset = "utf8mb4"
	}
}

func (c *Config) AfterInject() {
	(*gormi.Config)(c).Init()
}

func (c *Config) Build(dialector gorm.Dialector) (*gorm.DB, error) {

	dbConfig := &c.Gorm
	dbConfig.NamingStrategy = c.NamingStrategy

	// 日志
	if c.EnableStdLogger {
		// 默认日志
		logger.Default = logger.New(stdlog.New(os.Stdout, "\r", stdlog.LstdFlags), c.Logger)
	} else {
		logger.Default = &loggeri.Logger{Logger: log.NoCallerLogger().Logger, Config: &c.Logger}
	}

	db, err := gorm.Open(dialector, dbConfig)
	if err != nil {
		return nil, err
	}

	if c.EnablePrometheus {
		if c.Type == dbi.Mysql {
			for _, pc := range c.PrometheusConfigs {
				c.Prometheus.MetricsCollector = append(c.Prometheus.MetricsCollector, &prometheus.MySQL{
					Prefix:        pc.Prefix,
					Interval:      pc.Interval,
					VariableNames: pc.VariableNames,
				})
			}

		}
		if c.Type == dbi.Postgres {
			for _, pc := range c.PrometheusConfigs {
				c.Prometheus.MetricsCollector = append(c.Prometheus.MetricsCollector, &prometheus.Postgres{
					Prefix:        pc.Prefix,
					Interval:      pc.Interval,
					VariableNames: pc.VariableNames,
				})
			}
		}
		err = db.Use(prometheus.New(c.Prometheus))
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
