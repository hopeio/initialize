/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"fmt"
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

// resolveLocalPaths turns the configured local config paths absolute so the
// watchers and the env-file lookup behave the same regardless of the directory
// the process was started in.
//
// It used to run from AfterInject. That hook has no error channel, and a failing
// path resolution must reach the caller as a Start error instead of exiting the
// process from inside a lifecycle hook. Resolving an already-absolute path is
// just a Clean, so calling it twice (before the file load and after the env
// decode) is safe.
func (c *EnvConfig) resolveLocalPaths() error {
	for i := range c.LocalConfig.Paths {
		abs, err := filepath.Abs(c.LocalConfig.Paths[i])
		if err != nil {
			return fmt.Errorf("initialize: resolve config path %s: %w", c.LocalConfig.Paths[i], err)
		}
		c.LocalConfig.Paths[i] = abs
	}
	return nil
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
func (a *App[C, D, CPtr, DPtr]) setRootConfig() error {
	format := a.RootConfig.ConfigCenter.Format
	confPath := a.RootConfig.ConfPath

	if err := a.Viper.Unmarshal(&a.RootConfig, decoderConfigOptions...); err != nil {
		return fmt.Errorf("initialize: decode root config: %w", err)
	}
	if err := a.applyFlagConfig("", &a.RootConfig); err != nil {
		return err
	}
	if a.RootConfig.ConfigCenter.Format == "" {
		a.RootConfig.ConfigCenter.Format = format
	}
	if a.RootConfig.Name == "" {
		a.RootConfig.Name = stringsx.CutPart(filepath.Base(os.Args[0]), ".")
	}
	if a.RootConfig.ConfPath != confPath {
		a.RootConfig.ConfPath = confPath
	}
	return nil
}

// writeEnvConfigTemplate dumps the assembled env section (plus the registered
// config centers) into ConfigTemplateDir, so an operator can start from a
// complete file. A no-op when the directory is not configured.
func (a *App[C, D, CPtr, DPtr]) writeEnvConfigTemplate(format string) error {
	if a.RootConfig.EnvConfig.ConfigTemplateDir == "" {
		return nil
	}
	confMap := make(map[string]any)
	if err := struct2Map(&a.RootConfig.BasicConfig, confMap); err != nil {
		return err
	}
	envMap := make(map[string]any)
	if err := struct2Map(&a.RootConfig.EnvConfig, envMap); err != nil {
		return err
	}
	confMap[a.RootConfig.Env] = envMap
	ccMap := make(map[string]any)
	if err := struct2Map(&a.RootConfig.EnvConfig.ConfigCenter, ccMap); err != nil {
		return err
	}
	envMap[fixedFieldNameConfigCenter] = ccMap
	for name, v := range a.registeredCenters() {
		cc := make(map[string]any)
		if err := struct2Map(v.Config(), cc); err != nil {
			return err
		}
		ccMap[name] = cc
	}
	encoder, err := codecRegistry.Encoder(format)
	if err != nil {
		return fmt.Errorf("initialize: config template encoder %s: %w", format, err)
	}
	data, err := encoder.Encode(confMap)
	if err != nil {
		return fmt.Errorf("initialize: encode config template: %w", err)
	}

	dir := a.RootConfig.EnvConfig.ConfigTemplateDir
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("initialize: create config template dir %s: %w", dir, err)
	}
	if dir[len(dir)-1] != '/' {
		dir += "/"
	}
	if err := os.WriteFile(dir+prefixConfigTemplate+format, data, 0644); err != nil {
		return fmt.Errorf("initialize: write config template: %w", err)
	}
	return nil
}

// setEnvConfig reads the environment-specific config section (keyed by RootConfig.Env),
// decodes it into EnvConfig, optionally connects to a config center, and writes a config template if requested.
func (a *App[C, D, CPtr, DPtr]) setEnvConfig() (err error) {
	if a.RootConfig.Env == "" {
		if a.RootConfig.ConfPath == "" {
			log.Warn("not found config file, use env and flag")
		} else {
			log.Warn("lack of env configuration, try single config file mode")
		}
		return nil
	}
	format := a.RootConfig.ConfigCenter.Format

	// The template is written on every exit path, including the early returns
	// below that only warn; a failed assembly stops before writing one.
	defer func() {
		if err != nil {
			return
		}
		err = a.writeEnvConfigTemplate(format)
	}()

	envConfig, ok := a.Viper.Get(a.RootConfig.Env).(map[string]any)
	if !ok {
		log.Warnf("lack of env configuration: %s", a.RootConfig.Env)
		return nil
	}
	if err := Decode(&a.RootConfig.EnvConfig, envConfig); err != nil {
		return fmt.Errorf("initialize: decode env config %s: %w", a.RootConfig.Env, err)
	}
	flagPrefix := strings.ToLower(a.RootConfig.Name)
	if err := a.applyFlagConfig(flagPrefix, &a.RootConfig.EnvConfig); err != nil {
		return err
	}
	if err := a.RootConfig.EnvConfig.resolveLocalPaths(); err != nil {
		return err
	}
	for i := range a.RootConfig.SkipInjectDaos {
		a.RootConfig.SkipInjectDaos[i] = strings.ToUpper(a.RootConfig.SkipInjectDaos[i])
	}
	var configCenter ConfigCenter
	if a.RootConfig.EnvConfig.ConfigCenter.Type != "" {
		if a.RootConfig.EnvConfig.ConfigCenter.Format == "" {
			log.Warnf("lack of config center format, support format:%v", viper.SupportedExts)
			return nil
		}
		a.Viper.SetConfigType(a.RootConfig.EnvConfig.ConfigCenter.Format)

		configCenter = a.getCenter(strings.ToLower(a.RootConfig.EnvConfig.ConfigCenter.Type))
		if configCenter == nil {
			log.Warnf("lack of registered config center type : %s", a.RootConfig.EnvConfig.ConfigCenter.Type)
			return nil
		}
		configCenterConfig, ok := a.Viper.Get(a.RootConfig.Env + ".configcenter." + a.RootConfig.EnvConfig.ConfigCenter.Type).(map[string]any)
		if !ok {
			log.Warn("lack of config center config")
			return nil
		}
		if err := applyTagDefaults(configCenter.Config()); err != nil {
			return err
		}
		if err := Decode(configCenter.Config(), configCenterConfig); err != nil {
			return fmt.Errorf("initialize: decode config center %s: %w", a.RootConfig.EnvConfig.ConfigCenter.Type, err)
		}

		if flagPrefix != "" {
			flagPrefix = flagPrefix + ".configcenter." + strings.ToLower(a.RootConfig.EnvConfig.ConfigCenter.Type)
		} else {
			flagPrefix = "configcenter." + strings.ToLower(a.RootConfig.EnvConfig.ConfigCenter.Type)
		}
		if err := a.applyFlagConfig(flagPrefix, configCenter.Config()); err != nil {
			return err
		}
		a.RootConfig.EnvConfig.ConfigCenter.ConfigCenter = configCenter
	}
	return nil
}
