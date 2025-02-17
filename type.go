/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"github.com/hopeio/initialize/rootconf"
	"reflect"
)

type beforeInject interface {
	BeforeInject()
}

type beforeInjectWithRoot interface {
	BeforeInjectWithRoot(*rootconf.RootConfig)
}

type afterInject interface {
	AfterInject()
}

type afterInjectWithRoot interface {
	AfterInjectWithRoot(*rootconf.RootConfig)
}

type afterInjectConfig interface {
	AfterInjectConfig()
}

type afterInjectConfigWithRoot interface {
	AfterInjectConfigWithRoot(*rootconf.RootConfig)
}

type Config interface {
	// 注入之前设置默认值
	beforeInject
	// 注入之后初始化
	afterInject
}

type Dao interface {
	beforeInject
	// 注入config后执行
	afterInjectConfig
	// 注入dao后执行
	afterInject
}

type EmbeddedPresets struct {
}

func (u *EmbeddedPresets) BeforeInject() {
}
func (u *EmbeddedPresets) AfterInjectConfig() {
}
func (u *EmbeddedPresets) AfterInject() {
}

var EmbeddedPresetsType = reflect.TypeOf((*EmbeddedPresets)(nil)).Elem()
