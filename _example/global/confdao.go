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
	"github.com/hopeio/initialize/contrib/mqtt"
)

var (
	Global = initialize.NewGlobal[config, dao](nacos.ConfigCenter)
)

type config struct {
	initialize.EmbeddedPresets
	// Custom application config.
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

// BeforeInject sets default values before config is unmarshaled.
func (c *config) BeforeInject() {
	c.Customize.Duration = timex.Day
}

// AfterInject normalizes the Duration to hour granularity on non-Windows platforms.
func (c *config) AfterInject() {
	if runtime.GOOS == "windows" {
	}

	c.Customize.Duration = timex.NormalizeDuration(c.Customize.Duration, time.Hour)
}

// dao holds all DAO (data access object) fields for the application.
type dao struct {
	initialize.EmbeddedPresets
	// Mqtt is the MQTT client connection.
	Mqtt mqtt.Client
}

// BeforeInject is a no-op satisfying the Dao interface.
func (d *dao) BeforeInject() {

}

// AfterInjectConfig is called after the Mqtt config is injected; logs a message.
func (d *dao) AfterInjectConfig() {
	fmt.Println("AfterInjectConfig called")
}

// AfterInject publishes a test MQTT message to verify connectivity.
func (d *dao) AfterInject() {
	if token := d.Mqtt.Publish("test", 0, false, "test"); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
}
