/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package main

import (
	"fmt"
	"github.com/hopeio/initialize"
	"github.com/hopeio/initialize/_example/confdao"
	"github.com/hopeio/initialize/conf_center/nacos"
)

func main() {
	defer initialize.Start(confdao.Conf, confdao.Dao, nacos.ConfigCenter)()
	fmt.Println(confdao.Conf)
}
