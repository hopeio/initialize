/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package apollo

import (
	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/env/config"
	"io"
	"strings"
)

var ConfigCenter = &Apollo{}

type Apollo struct {
	Conf   config.AppConfig
	Client agollo.Client
}

func (e *Apollo) Type() string {
	return "apollo"
}

func (cc *Apollo) Config() any {
	return &cc.Conf
}

// TODD: 更改监听
func (e *Apollo) Handle(handle func(io.Reader)) error {
	var err error
	if e.Client == nil {
		e.Client, err = agollo.StartWithConfig(func() (*config.AppConfig, error) {
			return &e.Conf, nil
		})
		if err != nil {
			return err
		}
	}

	config := e.Client.GetConfig(e.Conf.NamespaceName)
	handle(strings.NewReader(config.GetContent()))
	return nil
}

func (cc *Apollo) Close() error {
	return nil
}
