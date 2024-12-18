/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package apollo

import (
	"encoding/json"
	"github.com/hopeio/utils/dao/apollo"
)

var ConfigCenter = &Apollo{}

type Apollo struct {
	Conf   apollo.Config
	Client *apollo.Client
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
		e.Client, err = e.Conf.NewClient()
		if err != nil {
			return err
		}
	}

	config, err := e.Client.GetDefaultConfig()
	if err != nil {
		return err
	}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	handle(data)
	return nil
}

func (cc *Apollo) Close() error {
	return nil
}
