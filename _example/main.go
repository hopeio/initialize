package main

import (
	"fmt"
	"github.com/hopeio/initialize"
	"github.com/hopeio/initialize/_example/confdao"
	"github.com/hopeio/initialize/conf_center/nacos"
)

func main() {
	defer initialize.Start(confdao.Conf, confdao.Dao, nacos.ConfigCenter)()
	fmt.Println(confdao.Conf.Server.Http.Addr)
}
