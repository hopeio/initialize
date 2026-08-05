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

// Type returns the identifier string "apollo" for this config center.
func (e *Apollo) Type() string {
	return "apollo"
}

// Config returns the embedded Conf as the configuration for injection.
func (cc *Apollo) Config() any {
	return &cc.Conf
}

// Handle starts the Apollo client, merges all configured namespaces, and registers a change listener.
// TODO: improve change-listener update handling.
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

// Close is a no-op; the Apollo client does not require explicit shutdown.
func (cc *Apollo) Close() error {
	return nil
}

// 1. 定义你的监听器结构体
type CustomListener struct {
	handle func(io.Reader) error
}

// OnChange is called when any Apollo config key changes; it serializes the changes as properties
// and passes them to the registered onChange handler.
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

// OnNewestChange is called with the full diff; currently a no-op.
func (l *CustomListener) OnNewestChange(event *storage.FullChangeEvent) {}
