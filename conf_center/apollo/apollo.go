/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package apollo

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/apolloconfig/agollo/v5"
	"github.com/apolloconfig/agollo/v5/env/config"
	"github.com/apolloconfig/agollo/v5/storage"
	"github.com/hopeio/gox/log"
)

var ConfigCenter = &Apollo{}

type Apollo struct {
	Conf   Config
	Client agollo.Client
}

type Config struct {
	config.AppConfig
	Namespaces []string
}

func (e *Apollo) Type() string {
	return "apollo"
}

func (cc *Apollo) Config() any {
	return &cc.Conf
}

// TODD: 更改监听
func (e *Apollo) Handle(ctx context.Context, merge func(io.Reader) error, onChange func(io.Reader) error) error {
	var err error
	if e.Client == nil {
		e.Client, err = agollo.StartWithConfig(func() (*config.AppConfig, error) {
			return &e.Conf.AppConfig, nil
		})
		if err != nil {
			return err
		}
	}

	for _, namespace := range e.Conf.Namespaces {
		config := e.Client.GetConfig(namespace)
		err = merge(strings.NewReader(config.GetContent()))
		if err != nil {
			return err
		}
	}

	e.Client.AddChangeListener(&CustomListener{handle: onChange})
	return nil
}

func (cc *Apollo) Close() error {
	return nil
}

// 1. 定义你的监听器结构体
type CustomListener struct {
	handle func(io.Reader) error
}

// 2. 实现 OnChange 方法
func (l *CustomListener) OnChange(event *storage.ChangeEvent) {

	properties := ""
	for key, value := range event.Changes {
		properties += fmt.Sprintf("%s=%v\n", key, value.NewValue)
	}
	err := l.handle(strings.NewReader(properties))
	if err != nil {
		log.Error(err)
	}
}

// 3. 实现 OnNewestChange 方法（通常留空或记录日志）
func (l *CustomListener) OnNewestChange(event *storage.FullChangeEvent) {}
