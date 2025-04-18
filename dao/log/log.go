/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package log

import (
	"github.com/hopeio/initialize/rootconf"
	"github.com/hopeio/utils/log"
	"go.uber.org/zap/zapcore"
	"reflect"
)

// 全局变量,只一个实例,只提供config
type Config log.Config

func (c *Config) AfterInjectWithRoot(rootconfig *rootconf.RootConfig) {
	isZero := reflect.ValueOf(c).Elem().IsZero()
	if rootconfig.Name != "" && c.Name == "" {
		c.Name = rootconfig.Name
		if isZero {
			c.Development = true
			c.Level = zapcore.DebugLevel
			isZero = false
		}
	}
	if !isZero {
		log.SetDefaultLogger((*log.Config)(c))
	}
}

func (c *Config) Build() *log.Logger {
	return (*log.Config)(c).NewLogger()
}

type Logger struct {
	*log.Logger
	Conf Config
}

func (l *Logger) Config() any {
	return &l.Conf
}

func (l *Logger) Init() error {
	l.Logger = l.Conf.Build()
	return nil
}

func (l *Logger) Close() error {
	if l.Logger == nil {
		return nil
	}
	return l.Logger.Sync()
}
