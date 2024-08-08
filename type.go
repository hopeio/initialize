package initialize

import (
	"github.com/hopeio/initialize/rootconf"
	"reflect"
)

type BeforeInject interface {
	BeforeInject()
}

type BeforeInjectWithRoot interface {
	BeforeInjectWithRoot(*rootconf.RootConfig)
}

type AfterInject interface {
	AfterInject()
}

type AfterInjectWithRoot interface {
	AfterInjectWithRoot(*rootconf.RootConfig)
}

type AfterInjectConfig interface {
	AfterInjectConfig()
}

type AfterInjectConfigWithRoot interface {
	AfterInjectConfigWithRoot(*rootconf.RootConfig)
}

type Config interface {
	// 注入之前设置默认值
	BeforeInject
	// 注入之后初始化
	AfterInject
}

type Dao interface {
	BeforeInject
	// 注入config后执行
	AfterInjectConfig
	// 注入dao后执行
	AfterInject
}

type EmbeddedPresets struct {
}

func (u EmbeddedPresets) BeforeInject() {
}
func (u EmbeddedPresets) AfterInjectConfig() {
}
func (u EmbeddedPresets) AfterInject() {
}

var EmbeddedPresetsType = reflect.TypeOf((*EmbeddedPresets)(nil)).Elem()
