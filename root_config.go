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
func (g *Global[C, D, CPtr, DPtr]) setRootConfig() {
	format := g.RootConfig.ConfigCenter.Format
	confPath := g.RootConfig.ConfPath

	err := g.Viper.Unmarshal(&g.RootConfig, decoderConfigOptions...)
	if err != nil {
		log.Fatal(err)
	}
	g.applyFlagConfig("", &g.RootConfig)
	if g.RootConfig.ConfigCenter.Format == "" {
		g.RootConfig.ConfigCenter.Format = format
	}
	if g.RootConfig.Name == "" {
		g.RootConfig.Name = stringsx.CutPart(filepath.Base(os.Args[0]), ".")
	}
	if g.RootConfig.ConfPath != confPath {
		g.RootConfig.ConfPath = confPath
	}
}

// setEnvConfig reads the environment-specific config section (keyed by RootConfig.Env),
// decodes it into EnvConfig, optionally connects to a config center, and writes a config template if requested.
func (g *Global[C, D, CPtr, DPtr]) setEnvConfig() {
	if g.RootConfig.Env == "" {
		if g.RootConfig.ConfPath == "" {
			log.Warn("not found config file, use env and flag")
		} else {
			log.Warn("lack of env configuration, try single config file mode")
		}
		return
	}
	format := g.RootConfig.ConfigCenter.Format

	defer func() {
		if g.RootConfig.EnvConfig.ConfigTemplateDir != "" {
			// template
			confMap := make(map[string]any)
			struct2Map(&g.RootConfig.BasicConfig, confMap)
			envMap := make(map[string]any)
			struct2Map(&g.RootConfig.EnvConfig, envMap)
			confMap[g.RootConfig.Env] = envMap
			ccMap := make(map[string]any)
			struct2Map(&g.RootConfig.EnvConfig.ConfigCenter, ccMap)
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

			dir := g.RootConfig.EnvConfig.ConfigTemplateDir
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

	envConfig, ok := g.Viper.Get(g.RootConfig.Env).(map[string]any)
	if !ok {
		log.Warnf("lack of env configuration: %s", g.RootConfig.Env)
		return
	}
	err := Decode(&g.RootConfig.EnvConfig, envConfig)
	if err != nil {
		log.Fatal(err)
	}
	flagPrefix := strings.ToLower(g.RootConfig.Name)
	g.applyFlagConfig(flagPrefix, &g.RootConfig.EnvConfig)
	g.RootConfig.EnvConfig.AfterInject()
	for i := range g.RootConfig.SkipInjectDaos {
		g.RootConfig.SkipInjectDaos[i] = strings.ToUpper(g.RootConfig.SkipInjectDaos[i])
	}
	var configCenter ConfigCenter
	if g.RootConfig.EnvConfig.ConfigCenter.Type != "" {
		if g.RootConfig.EnvConfig.ConfigCenter.Format == "" {
			log.Warnf("lack of config center format, support format:%v", viper.SupportedExts)
			return
		}
		g.Viper.SetConfigType(g.RootConfig.EnvConfig.ConfigCenter.Format)

		configCenter, ok = GetRegisteredConfigCenter()[strings.ToLower(g.RootConfig.EnvConfig.ConfigCenter.Type)]
		if !ok {
			log.Warnf("lack of registered config center type : %s", g.RootConfig.EnvConfig.ConfigCenter.Type)
			return
		}
		configCenterConfig, ok := g.Viper.Get(g.RootConfig.Env + ".configcenter." + g.RootConfig.EnvConfig.ConfigCenter.Type).(map[string]any)
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
			flagPrefix = flagPrefix + ".configcenter." + strings.ToLower(g.RootConfig.EnvConfig.ConfigCenter.Type)
		} else {
			flagPrefix = "configcenter." + strings.ToLower(g.RootConfig.EnvConfig.ConfigCenter.Type)
		}
		g.applyFlagConfig(flagPrefix, configCenter.Config())
		g.RootConfig.EnvConfig.ConfigCenter.ConfigCenter = configCenter
	}
}
