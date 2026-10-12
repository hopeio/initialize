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

	"github.com/hopeio/gox/log"
	stringsx "github.com/hopeio/gox/strings"
	"github.com/spf13/viper"
)

type RootConfig struct {
	Executable string `init:"-"` // autowired
	ExecDir    string `init:"-"` // autowired
	// Config file path
	ConfPath string `init:"flag:config;short_flag:c;usage:config file path, default ./config.xxx or ./config/config.xxx;env:CONFIG"`
	BasicConfig
	EnvConfig
}

// BasicConfig holds module identity fields shared across environments.
type BasicConfig struct {
	// Module name
	Name string `init:"flag:name;usage:module name;env:NAME"`
	// environment
	Env string `init:"flag:env;short_flag:e;default:dev;usage:environment;env:ENV"`
}

type EnvConfig struct {
	Debug             bool     `init:"flag:debug;short_flag:d;usage:enable debug mode"`
	ConfigTemplateDir string   `init:"flag:conf_tmpl_dir;usage:directory to write config templates"`
	SkipInjectDaos    []string `init:"flag:skip_inject_daos;usage:dao names to skip during injection"`
	LocalConfig       Local
	// Field order must stay unchanged; ConfigCenter must remain last.
	ConfigCenter ConfigCenterConfig
}

// AfterInject sets up the HTTP proxy from the Proxy field and resolves all local config paths to absolute paths.
func (c *EnvConfig) AfterInject() {
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

// setRootConfig unmarshals the top-level (squash) keys from Viper into RootConfig,
// applies flag overrides, and fills in defaults for Name and ConfPath.
func (a *App[C, D, CPtr, DPtr]) setRootConfig() {
	format := a.RootConfig.ConfigCenter.Format
	confPath := a.RootConfig.ConfPath

	err := a.Viper.Unmarshal(&a.RootConfig, decoderConfigOptions...)
	if err != nil {
		log.Fatal(err)
	}
	a.applyFlagConfig("", &a.RootConfig)
	if a.RootConfig.ConfigCenter.Format == "" {
		a.RootConfig.ConfigCenter.Format = format
	}
	if a.RootConfig.Name == "" {
		a.RootConfig.Name = stringsx.CutPart(filepath.Base(os.Args[0]), ".")
	}
	if a.RootConfig.ConfPath != confPath {
		a.RootConfig.ConfPath = confPath
	}
}

// setEnvConfig reads the environment-specific config section (keyed by RootConfig.Env),
// decodes it into EnvConfig, optionally connects to a config center, and writes a config template if requested.
func (a *App[C, D, CPtr, DPtr]) setEnvConfig() {
	if a.RootConfig.Env == "" {
		if a.RootConfig.ConfPath == "" {
			log.Warn("not found config file, use env and flag")
		} else {
			log.Warn("lack of env configuration, try single config file mode")
		}
		return
	}
	format := a.RootConfig.ConfigCenter.Format

	defer func() {
		if a.RootConfig.EnvConfig.ConfigTemplateDir != "" {
			// template
			confMap := make(map[string]any)
			struct2Map(&a.RootConfig.BasicConfig, confMap)
			envMap := make(map[string]any)
			struct2Map(&a.RootConfig.EnvConfig, envMap)
			confMap[a.RootConfig.Env] = envMap
			ccMap := make(map[string]any)
			struct2Map(&a.RootConfig.EnvConfig.ConfigCenter, ccMap)
			envMap[fixedFieldNameConfigCenter] = ccMap
			if cc := a.RootConfig.ConfigCenter.ConfigCenter; cc != nil {
				ccValues := make(map[string]any)
				struct2Map(cc.Config(), ccValues)
				ccMap[cc.Type()] = ccValues
			}
			endocer, err := codecRegistry.Encoder(format)
			if err != nil {
				log.Fatal(err)
			}
			data, err := endocer.Encode(confMap)
			if err != nil {
				log.Fatal(err)
			}

			dir := a.RootConfig.EnvConfig.ConfigTemplateDir
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

	envConfig, ok := a.Viper.Get(a.RootConfig.Env).(map[string]any)
	if !ok {
		log.Warnf("lack of env configuration: %s", a.RootConfig.Env)
		return
	}
	err := Decode(&a.RootConfig.EnvConfig, envConfig)
	if err != nil {
		log.Fatal(err)
	}
	flagPrefix := strings.ToLower(a.RootConfig.Name)
	a.applyFlagConfig(flagPrefix, &a.RootConfig.EnvConfig)
	a.RootConfig.EnvConfig.AfterInject()
	for i := range a.RootConfig.SkipInjectDaos {
		a.RootConfig.SkipInjectDaos[i] = strings.ToUpper(a.RootConfig.SkipInjectDaos[i])
	}
	// The config center is injected via Start's parameter (already stored in
	// the field above); the env config section only carries its own settings.
	if a.RootConfig.EnvConfig.ConfigCenter.Type != "" {
		if a.RootConfig.EnvConfig.ConfigCenter.Format == "" {
			log.Warnf("lack of config center format, support format:%v", viper.SupportedExts)
			return
		}
		a.Viper.SetConfigType(a.RootConfig.EnvConfig.ConfigCenter.Format)

		configCenter := a.RootConfig.ConfigCenter.ConfigCenter
		if configCenter == nil {
			log.Warnf("no config center passed to Start for type: %s", a.RootConfig.EnvConfig.ConfigCenter.Type)
			return
		}
		configCenterConfig, ok := a.Viper.Get(a.RootConfig.Env + ".configcenter." + a.RootConfig.EnvConfig.ConfigCenter.Type).(map[string]any)
		if !ok {
			log.Warn("lack of config center config")
			return
		}
		applyTagDefaults(configCenter.Config())
		err = Decode(configCenter.Config(), configCenterConfig)
		if err != nil {
			log.Fatal(err)
		}

		if flagPrefix != "" {
			flagPrefix = flagPrefix + ".configcenter." + strings.ToLower(a.RootConfig.EnvConfig.ConfigCenter.Type)
		} else {
			flagPrefix = "configcenter." + strings.ToLower(a.RootConfig.EnvConfig.ConfigCenter.Type)
		}
		a.applyFlagConfig(flagPrefix, configCenter.Config())
	}
}
