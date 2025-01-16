/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package groupcache

import (
	"github.com/golang/groupcache"
)

type Config struct {
	Name       string
	CacheBytes int64
	groupcache.GetterFunc
}

func (c *Config) BeforeInject() {

}

func (c *Config) AfterInject() {
}

func (c *Config) Build() *groupcache.Group {
	return groupcache.NewGroup(c.Name, c.CacheBytes, c.GetterFunc)
}

type Group struct {
	*groupcache.Group
	Conf Config
}

func (m *Group) Config() any {
	return &m.Conf
}

func (m *Group) Init() error {
	m.Group = m.Conf.Build()
	return nil
}

func (m *Group) Close() error {
	return nil
}
