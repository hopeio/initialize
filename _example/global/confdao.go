/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package global

import (
	"fmt"
	"runtime"
	"time"

	timex "github.com/hopeio/gox/time"
	"github.com/hopeio/initialize"
	"github.com/hopeio/initialize/conf_center/nacos"
	"github.com/hopeio/initialize/dao/mqtt"
)

var (
	Global = initialize.NewGlobal[*config, *dao](nacos.ConfigCenter)
)

type config struct {
	initialize.EmbeddedPresets
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
	c.Customize.Duration = timex.Day
}

func (c *config) AfterInject() {
	if runtime.GOOS == "windows" {
	}

	c.Customize.Duration = timex.StdDuration(c.Customize.Duration, time.Hour)
}

// dao dao.
type dao struct {
	initialize.EmbeddedPresets
	// GORMDB 数据库连接
	Mqtt mqtt.Client
}

func (d *dao) BeforeInject() {

}

func (d *dao) AfterInjectConfig() {
	fmt.Println("这里后执行")
}

func (d *dao) AfterInject() {
	if token := d.Mqtt.Publish("test", 0, false, "test"); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
}
