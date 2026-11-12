/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package gormdb

import (
	"testing"

	"github.com/hopeio/gox/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm/logger"
)

func TestBuildGormLoggerIncludesComponent(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)
	previousLogger := log.DefaultLogger()
	log.SetDefaultLogger(&log.Logger{Logger: zap.New(core)})
	previousDefault := logger.Default
	t.Cleanup(func() {
		log.SetDefaultLogger(previousLogger)
		logger.Default = previousDefault
	})

	conf := &Config{
		MaxOpenConns: 1,
		Logger:       logger.Config{LogLevel: logger.Info},
	}
	db, err := conf.Build(sqlite.Open(":memory:"))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	if err := db.Exec("SELECT 1").Error; err != nil {
		t.Fatalf("Exec: %v", err)
	}
	entries := observed.FilterField(zap.String("component", "gorm")).All()
	if len(entries) != 1 {
		t.Fatalf("gorm component logs = %d, want 1; all logs: %+v", len(entries), observed.All())
	}
	if got := entries[0].ContextMap()["sql"]; got != "SELECT 1" {
		t.Fatalf("sql = %v, want SELECT 1", got)
	}
	if logger.Default != db.Logger {
		t.Fatal("logger.Default should be the same logger installed on the DB")
	}
}
