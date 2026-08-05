/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"reflect"

	"github.com/hopeio/gox/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type builtinConfig struct {
	Log LogConfig
}

// 全局变量,只一个实例,只提供config
type LogConfig log.Config

func (c *LogConfig) AfterInjectWithRoot(rootconfig *RootConfig) {
	isZero := reflect.ValueOf(c).Elem().IsZero()
	if rootconfig.Name != "" && c.Name == "" {
		c.Name = rootconfig.Name
		if isZero {
			c.Development = true
			c.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
			isZero = false
		}
	}
	if !isZero {
		c.Name = rootconfig.Name
		c.Development = rootconfig.EnvConfig.Debug
		logger := (*log.Config)(c).NewLogger()
		log.SetDefaultLogger(logger)
	}
}
