/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"github.com/hopeio/initialize/conf_center"
	"github.com/hopeio/utils/log"
	"github.com/hopeio/utils/reflect/mtos"
	"github.com/spf13/viper"
	"os"
	"strings"
)

const (
	fixedFieldNameEnvConfig       = "EnvConfig"
	fixedFieldNameBasicConfig     = "RootConfig"
	fixedFieldNameConfigCenter    = "ConfigCenter"
	fixedFieldNameEnv             = "Env"
	fixedFieldNameEncoderRegistry = "encoderRegistry"
	prefixConfigTemplate          = "config.template."
	prefixLocalTemplate           = "local.template."
	skipTypeTlsConfig             = "tls.Config"
)

func (gc *globalConfig[C, D]) setEnvConfig() {
	if gc.RootConfig.Env == "" {
		if gc.RootConfig.ConfPath == "" {
			log.Warn("not found config, use env and flag")
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
			for name, v := range conf_center.GetRegisteredConfigCenter() {
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
		if gc.RootConfig.ConfPath == "" {
			log.Warn("not found config, use env and flag")
		} else {
			log.Warn("lack of env configuration, try single config file mode")
		}
		return
	}
	err := mtos.Unmarshal(&gc.RootConfig.EnvConfig, envConfig)
	if err != nil {
		log.Fatal(err)
	}
	applyFlagConfig(nil, &gc.RootConfig.EnvConfig)
	gc.RootConfig.EnvConfig.AfterInject()
	if gc.RootConfig.EnvConfig.ConfigCenter.Format == "" {
		log.Fatalf("lack of configCenter format, support format:%v", viper.SupportedExts)
	}
	gc.Viper.SetConfigType(gc.RootConfig.EnvConfig.ConfigCenter.Format)
	if gc.RootConfig.EnvConfig.ConfigCenter.Type == "" {
		log.Warn("lack of configCenter type, try single config file")
		return
	}

	configCenter, ok := conf_center.GetRegisteredConfigCenter()[strings.ToLower(gc.RootConfig.EnvConfig.ConfigCenter.Type)]
	if !ok {
		log.Warn("lack of registered configCenter, try single config file")
		return
	}

	applyFlagConfig(gc.Viper, configCenter.Config())
	gc.RootConfig.EnvConfig.ConfigCenter.ConfigCenter = configCenter

	configCenterConfig, ok := gc.Viper.Get(gc.RootConfig.Env + ".configCenter." + gc.RootConfig.EnvConfig.ConfigCenter.Type).(map[string]any)
	if !ok {
		log.Warn("lack of configCenter config, try single config file")
		return
	}
	err = mtos.Unmarshal(configCenter.Config(), configCenterConfig)
	if err != nil {
		log.Fatal(err)
	}
}
