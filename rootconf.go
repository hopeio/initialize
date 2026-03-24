/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hopeio/gox/kvstruct"
	"github.com/hopeio/gox/log"
	stringsx "github.com/hopeio/gox/strings"
	"github.com/hopeio/initialize/conf_center"
	"github.com/spf13/viper"
)

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
	err := kvstruct.Unmarshal(&gc.RootConfig.EnvConfig, envConfig)
	if err != nil {
		log.Fatal(err)
	}
	flagPrefix := strings.ToLower(gc.RootConfig.Name)
	gc.applyFlagConfig(flagPrefix, &gc.RootConfig.EnvConfig)
	gc.RootConfig.EnvConfig.AfterInject()
	for i := range gc.RootConfig.SkipInjectDaos {
		gc.RootConfig.SkipInjectDaos[i] = strings.ToUpper(gc.RootConfig.SkipInjectDaos[i])
	}
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
		configCenterConfig, ok := gc.Viper.Get(gc.RootConfig.Env + ".configcenter." + gc.RootConfig.EnvConfig.ConfigCenter.Type).(map[string]any)
		if !ok {
			log.Warn("lack of config center config")
			return
		}
		err = kvstruct.Unmarshal(configCenter.Config(), configCenterConfig)
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
