/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package conf_center

import (
	"context"
	"io"
	"strings"
	"sync"

	"github.com/hopeio/gox/log"
	stringsx "github.com/hopeio/gox/text/encoding/ascii"
)

type ConfigType string

type ConfigCenter interface {
	Config() any
	io.Closer
	Handle(ctx context.Context, merge func(io.Reader) error, onChange func(io.Reader) error) error
	Type() string
}

type Config struct {
	// 配置格式
	Format string `flag:"name:format;usage:配置格式"`
	// 配置类型
	Type string `flag:"name:conf_type;usage:配置类型"`
	// config字段顺序不能变,ConfigCenter 保持在最后
	ConfigCenter ConfigCenter
}

var (
	configCenterMu sync.RWMutex
	configCenter   = map[string]ConfigCenter{}
)

func RegisterConfigCenter(c ConfigCenter) {
	if c == nil {
		return
	}
	typ := strings.ToLower(c.Type())
	if !stringsx.IsAllLetter(typ) {
		log.Fatal("config type must be letters")
	}
	configCenterMu.Lock()
	defer configCenterMu.Unlock()
	if _, ok := configCenter[typ]; !ok {
		configCenter[typ] = c
	}
}

func GetConfigCenter(configType string) ConfigCenter {
	configCenterMu.RLock()
	defer configCenterMu.RUnlock()
	return configCenter[configType]
}

func GetRegisteredConfigCenter() map[string]ConfigCenter {
	configCenterMu.RLock()
	defer configCenterMu.RUnlock()
	cp := make(map[string]ConfigCenter, len(configCenter))
	for k, v := range configCenter {
		cp[k] = v
	}
	return cp
}

type Client interface {
	Get() ([]byte, error)
	Set(func([]byte)) error
	Listener(func([]byte)) error
}
