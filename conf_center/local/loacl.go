/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package local

import (
	"errors"
	"fmt"
	"github.com/hopeio/utils/os/fs/loader"
	"os"
)

var ConfigCenter = &Local{}

type Local struct {
	loader.Loader
	ConfigPath string
}

func (cc *Local) Type() string {
	return "local"
}

func (cc *Local) Config() any {
	return cc
}

// 本地配置
func (cc *Local) Handle(handle func([]byte)) error {
	if cc.ConfigPath == "" {
		return errors.New("empty local config path")
	}
	_, err := os.Stat(cc.ConfigPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("找不到配置: %v", err)
	}

	err = cc.Loader.Handle(handle, cc.ConfigPath)
	if err != nil {
		return fmt.Errorf("配置错误: %v", err)
	}

	return nil
}

func (cc *Local) Close() error {
	return nil
}
