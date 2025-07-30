/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"github.com/hopeio/initialize/conf_center"
	"github.com/hopeio/gox/log"
	"github.com/hopeio/gox/reflect/mtos"
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
		log.Warnf("lack of env configuration: %s", gc.RootConfig.Env)
		return
	}
	err := mtos.Unmarshal(&gc.RootConfig.EnvConfig, envConfig)
	if err != nil {
		log.Fatal(err)
	}
	applyFlagConfig(nil, &gc.RootConfig.EnvConfig)
	gc.RootConfig.EnvConfig.AfterInject()

	var configCenter conf_center.ConfigCenter
	if gc.RootConfig.EnvConfig.ConfigCenter.Type != "" {

		if gc.RootConfig.EnvConfig.ConfigCenter.Format == "" {
			log.Warnf("lack of config center format, support format:%v", viper.SupportedExts)
			return
		}
		gc.Viper.SetConfigType(gc.RootConfig.EnvConfig.ConfigCenter.Format)

		configCenter, ok = conf_center.GetRegisteredConfigCenter()[strings.ToLower(gc.RootConfig.EnvConfig.ConfigCenter.Type)]
		if !ok {
			log.Warnf("lack of registered config center type : %s", gc.RootConfig.EnvConfig.ConfigCenter.Type)
			return
		}
		applyFlagConfig(gc.Viper, configCenter.Config())
		gc.RootConfig.EnvConfig.ConfigCenter.ConfigCenter = configCenter
		configCenterConfig, ok := gc.Viper.Get(gc.RootConfig.Env + ".configCenter." + gc.RootConfig.EnvConfig.ConfigCenter.Type).(map[string]any)
		if !ok {
			log.Warn("lack of config center config")
			return
		}
		err = mtos.Unmarshal(configCenter.Config(), configCenterConfig)
		if err != nil {
			log.Fatal(err)
		}
	}
}
