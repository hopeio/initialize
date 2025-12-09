/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package rootconf

import (
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/hopeio/gox/log"
	"github.com/hopeio/initialize/conf_center"
)

type RootConfig struct {
	Executable string
	ExecDir    string
	// 配置文件路径
	ConfPath string `flag:"name:config;short:c;usage:配置文件路径,默认./config.xxx或./config/config.xxx;env:CONFIG"`
	BasicConfig
	EnvConfig
}

// BasicConfig
type BasicConfig struct {
	// 模块名
	Name string `flag:"name:name;usage:模块名;env:APP_NAME"`
	// environment
	Env string `flag:"name:env;short:e;default:dev;usage:环境;env:ENV"`
}

type EnvConfig struct {
	Debug             bool   `flag:"name:debug;short:d;default:true;usage:是否测试;env:DEBUG"`
	ConfigTemplateDir string `flag:"name:conf_tmpl_dir;usage:是否生成配置模板;env:CONFIG_TEMPLATE_DIR"`
	// 代理, socks5://localhost:1080
	Proxy       string `flag:"name:proxy;usage:代理;env:HTTP_PROXY" `
	NoInject    []string
	LocalConfig conf_center.Local
	// config字段顺序不能变,ConfigCenter 保持在最后
	ConfigCenter conf_center.Config
}

func (c *EnvConfig) AfterInject() {
	if c.Proxy != "" {
		proxyURL, err := url.Parse(c.Proxy)
		if err != nil {
			log.Fatal(err)
		}
		http.DefaultClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	}
	var err error
	for i := range c.LocalConfig.Paths {
		c.LocalConfig.Paths[i], err = filepath.Abs(c.LocalConfig.Paths[i])
		if err != nil {
			log.Fatal(err)
		}
	}
}
