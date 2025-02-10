/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package apollo

import (
	"encoding/json"
	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/env/config"
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
func (e *Apollo) Handle(handle func([]byte)) error {
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
	data, err := json.Marshal(config.GetContent())
	if err != nil {
		return err
	}
	handle(data)
	return nil
}

func (cc *Apollo) Close() error {
	return nil
}
