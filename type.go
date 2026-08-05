/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"reflect"
)

type Init interface {
	Init()
}

type beforeInject interface {
	BeforeInject()
}

type beforeInjectWithRoot interface {
	BeforeInjectWithRoot(*RootConfig)
}

type afterInject interface {
	AfterInject()
}

type afterInjectWithRoot interface {
	AfterInjectWithRoot(*RootConfig)
}

type afterInjectConfig interface {
	AfterInjectConfig()
}

type afterInjectConfigWithRoot interface {
	AfterInjectConfigWithRoot(*RootConfig)
}

// Config is the interface that application configs must implement.
// BeforeInject sets default values before unmarshaling; AfterInject runs post-unmarshal initialization.
type Config interface {
	beforeInject
	afterInject
}

// Dao is the interface that DAO structs must implement.
// AfterInjectConfig runs after config is unmarshaled; AfterInject runs after all DAO fields are initialized.
type Dao interface {
	beforeInject
	afterInjectConfig
	afterInject
}

type EmbeddedPresets struct {
}

// BeforeInject is a no-op placeholder satisfying the Config/Dao interface.
func (u *EmbeddedPresets) BeforeInject() {
}

// AfterInjectConfig is a no-op placeholder satisfying the Dao interface.
func (u *EmbeddedPresets) AfterInjectConfig() {
}

// AfterInject is a no-op placeholder satisfying the Config/Dao interface.
func (u *EmbeddedPresets) AfterInject() {
}

var EmbeddedPresetsType = reflect.TypeOf((*EmbeddedPresets)(nil)).Elem()
