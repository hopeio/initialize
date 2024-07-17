package server

import "github.com/hopeio/cherry"

// 全局变量,只一个实例,只提供config
type Config cherry.Config

func (c *Config) InitBeforeInject() {
	*c = Config(*cherry.NewConfig())
}
func (c *Config) InitAfterInject() {
	(*cherry.Config)(c).Init()
}

// TODO: 是否会随着配置而更新
func (c *Config) Update() bool {
	return false
}

func (c *Config) Origin() *cherry.Config {
	return (*cherry.Config)(c)
}
