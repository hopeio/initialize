/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package main

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/hopeio/initialize/_example/global"
	"github.com/hopeio/initialize/conf_center/nacos"
)

func main() {
	// The config center is passed as a Start argument: it is registered before
	// assembly and picked up by the [ConfigCenter].Type in the config file.
	defer global.Global.Start(nacos.ConfigCenter)()
	global.Global.Defer(func() {
		fmt.Println("defer")
	})
	spew.Dump(global.Global.Dao)
	spew.Dump(global.Global.Conf())
	fmt.Println(global.Global.RootConfig.Env)
}
