/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package confdao

import (
	"database/sql"
	"fmt"
	"github.com/hopeio/initialize/conf_dao/gormdb/sqlite"
	timei "github.com/hopeio/utils/time"
	"runtime"
	"time"
)

var (
	Conf      = &config{}
	Dao  *dao = &dao{}
)

type config struct {
	//自定义的配置
	Customize serverConfig
}

type serverConfig struct {
	Int    int
	Float  float64
	String string
	Bool   bool
	Time   time.Time
	time.Duration
}

func (c *config) BeforeInject() {
	c.Customize.Duration = timei.Day
}

func (c *config) AfterInject() {
	if runtime.GOOS == "windows" {
	}

	c.Customize.Duration = timei.StdDuration(c.Customize.Duration, time.Hour)
}

// dao dao.
type dao struct {
	// GORMDB 数据库连接
	GORMDB sqlite.DB
	StdDB  *sql.DB
}

func (d *dao) BeforeInject() {
	d.GORMDB.Conf.Gorm.NowFunc = time.Now
}

func (d *dao) AfterInjectConfig() {
	fmt.Println("这里后执行")
}

func (d *dao) AfterInject() {
	db := d.GORMDB
	db.Callback().Create().Remove("gorm:save_before_associations")
	db.Callback().Create().Remove("gorm:save_after_associations")
	db.Callback().Update().Remove("gorm:save_before_associations")
	db.Callback().Update().Remove("gorm:save_after_associations")

	d.StdDB, _ = db.DB.DB()
}
