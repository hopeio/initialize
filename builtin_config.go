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

// Singleton-style wrapper: one global logger config instance.
type LogConfig log.Config

// AfterInjectWithRoot configures and replaces the default global logger using the root config's
// Name and Debug fields. It is a no-op if the config is still fully zero-valued.
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
