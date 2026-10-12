/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package main

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/hopeio/initialize"
	"github.com/hopeio/initialize/_example/global"
	"github.com/hopeio/initialize/conf_center/nacos"
)

func main() {
	// Register custom config centers before Start; the center configured in
	// the config file ([ConfigCenter].Type) is resolved from this registry.
	initialize.RegisterConfigCenter(nacos.ConfigCenter)
	defer global.Global.Start()()
	global.Global.Defer(func() {
		fmt.Println("defer")
	})
	spew.Dump(global.Global.Dao)
	spew.Dump(global.Global.Conf())
	fmt.Println(global.Global.RootConfig.Env)
}
