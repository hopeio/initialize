/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/hopeio/gox/log"
	stringsx "github.com/hopeio/gox/strings"
	"github.com/spf13/viper"
)

type RootConfig struct {
	Executable string `init:"-"` // autowired
	ExecDir    string `init:"-"` // autowired
	// 配置文件路径
	ConfPath string `flag:"name:config;short:c;usage:配置文件路径,默认./config.xxx或./config/config.xxx;env:CONFIG"`
	BasicConfig
	EnvConfig
}

// BasicConfig
type BasicConfig struct {
	// 模块名
	Name string `flag:"name:name;usage:模块名;env:NAME"`
	// environment
	Env string `flag:"name:env;short:e;default:dev;usage:环境;env:ENV"`
}

type EnvConfig struct {
	Debug             bool   `flag:"name:debug;short:d;default:true;usage:是否测试;env:DEBUG"`
	ConfigTemplateDir string `flag:"name:conf_tmpl_dir;usage:是否生成配置模板;env:CONFIG_TEMPLATE_DIR"`
	// 代理, socks5://localhost:1080
	Proxy          string   `flag:"name:proxy;usage:代理;env:HTTP_PROXY" `
	SkipInjectDaos []string `flag:"name:skip_inject_daos;usage:跳过注入的dao"`
	LocalConfig    Local
	// config字段顺序不能变,ConfigCenter 保持在最后
	ConfigCenter ConfigCenterConfig
}

func (c *EnvConfig) AfterInject() {
	if c.Proxy != "" {
		proxyURL, err := url.Parse(c.Proxy)
		if err != nil {
			log.Errorf("invalid proxy url %q, ignore it: %v", c.Proxy, err)
		} else {
			http.DefaultClient.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
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

const (
	fixedFieldNameEnvConfig       = "EnvConfig"
	fixedFieldNameBasicConfig     = "RootConfig"
	fixedFieldNameConfigCenter    = "ConfigCenter"
	fixedFieldNameEnv             = "Env"
	fixedFieldNameEncoderRegistry = "encoderRegistry"
	prefixConfigTemplate          = "config.template."
	prefixLocalTemplate           = "local.template."
)

func (gc *globalConfig[C, D]) setRootConfig() {
	format := gc.RootConfig.ConfigCenter.Format
	confPath := gc.RootConfig.ConfPath

	err := gc.Viper.Unmarshal(&gc.RootConfig, decoderConfigOptions...)
	if err != nil {
		log.Fatal(err)
	}
	gc.applyFlagConfig("", &gc.RootConfig)
	if gc.RootConfig.ConfigCenter.Format == "" {
		gc.RootConfig.ConfigCenter.Format = format
	}
	if gc.RootConfig.Name == "" {
		gc.RootConfig.Name = stringsx.CutPart(filepath.Base(os.Args[0]), ".")
	}
	if gc.RootConfig.ConfPath != confPath {
		gc.RootConfig.ConfPath = confPath
	}
}

func (gc *globalConfig[C, D]) setEnvConfig() {
	if gc.RootConfig.Env == "" {
		if gc.RootConfig.ConfPath == "" {
			log.Warn("not found config file, use env and flag")
		} else {
			log.Warn("lack of env configuration, try single config file mode")
		}
		return
	}
	format := gc.RootConfig.ConfigCenter.Format

	defer func() {
		if gc.RootConfig.EnvConfig.ConfigTemplateDir != "" {
			// template
			confMap := make(map[string]any)
			struct2Map(&gc.RootConfig.BasicConfig, confMap)
			envMap := make(map[string]any)
			struct2Map(&gc.RootConfig.EnvConfig, envMap)
			confMap[gc.RootConfig.Env] = envMap
			ccMap := make(map[string]any)
			struct2Map(&gc.RootConfig.EnvConfig.ConfigCenter, ccMap)
			envMap[fixedFieldNameConfigCenter] = ccMap
			for name, v := range GetRegisteredConfigCenter() {
				cc := make(map[string]any)
				struct2Map(v.Config(), cc)
				ccMap[name] = cc
			}
			endocer, err := codecRegistry.Encoder(format)
			if err != nil {
				log.Fatal(err)
			}
			data, err := endocer.Encode(confMap)
			if err != nil {
				log.Fatal(err)
			}

			dir := gc.RootConfig.EnvConfig.ConfigTemplateDir
			err = os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				log.Fatal(err)
			}
			if dir[len(dir)-1] != '/' {
				dir += "/"
			}
			err = os.WriteFile(dir+prefixConfigTemplate+format, data, 0644)
			if err != nil {
				log.Fatal(err)
			}
		}
	}()

	envConfig, ok := gc.Viper.Get(gc.RootConfig.Env).(map[string]any)
	if !ok {
		log.Warnf("lack of env configuration: %s", gc.RootConfig.Env)
		return
	}
	err := Decode(&gc.RootConfig.EnvConfig, envConfig)
	if err != nil {
		log.Fatal(err)
	}
	flagPrefix := strings.ToLower(gc.RootConfig.Name)
	gc.applyFlagConfig(flagPrefix, &gc.RootConfig.EnvConfig)
	gc.RootConfig.EnvConfig.AfterInject()
	for i := range gc.RootConfig.SkipInjectDaos {
		gc.RootConfig.SkipInjectDaos[i] = strings.ToUpper(gc.RootConfig.SkipInjectDaos[i])
	}
	var configCenter ConfigCenter
	if gc.RootConfig.EnvConfig.ConfigCenter.Type != "" {
		if gc.RootConfig.EnvConfig.ConfigCenter.Format == "" {
			log.Warnf("lack of config center format, support format:%v", viper.SupportedExts)
			return
		}
		gc.Viper.SetConfigType(gc.RootConfig.EnvConfig.ConfigCenter.Format)

		configCenter, ok = GetRegisteredConfigCenter()[strings.ToLower(gc.RootConfig.EnvConfig.ConfigCenter.Type)]
		if !ok {
			log.Warnf("lack of registered config center type : %s", gc.RootConfig.EnvConfig.ConfigCenter.Type)
			return
		}
		configCenterConfig, ok := gc.Viper.Get(gc.RootConfig.Env + ".configcenter." + gc.RootConfig.EnvConfig.ConfigCenter.Type).(map[string]any)
		if !ok {
			log.Warn("lack of config center config")
			return
		}
		err = Decode(configCenter.Config(), configCenterConfig)
		if err != nil {
			log.Fatal(err)
		}

		if flagPrefix != "" {
			flagPrefix = flagPrefix + ".configcenter." + strings.ToLower(gc.RootConfig.EnvConfig.ConfigCenter.Type)
		} else {
			flagPrefix = "configcenter." + strings.ToLower(gc.RootConfig.EnvConfig.ConfigCenter.Type)
		}
		gc.applyFlagConfig(flagPrefix, configCenter.Config())
		gc.RootConfig.EnvConfig.ConfigCenter.ConfigCenter = configCenter
	}
}
