package dingtalk

import (
	"github.com/hopeio/gox/sdk/dingtalk"
)

type Config dingtalk.Robot

func (c *Config) BeforeInject() {

}

func (c *Config) AfterInject() {
}

type Robot struct {
	dingtalk.Robot
}

func (m *Robot) Config() any {
	return &m.Robot
}

func (m *Robot) Init() error {
	return nil
}

func (m *Robot) Close() error {
	return nil
}
