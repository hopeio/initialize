/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package main

import (
	"fmt"
	"github.com/hopeio/initialize/_example/global"
)

func main() {
	defer global.Global.Cleanup()
	global.Global.Defer(func() {
		fmt.Println("defer")
	})
	fmt.Println(global.Global.Config)
	fmt.Println(global.Global.RootConfig.Env)
}
