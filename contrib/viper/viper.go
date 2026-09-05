/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package viper

import (
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
	"gopkg.in/ini.v1"
	"os"
)

// Config holds Viper settings for a single shared instance (config-only).
type Config struct {
	Debug             bool
	Watch             bool
	ConfigName        string
	ConfigFile        string
	ConfigType        string
	ConfigPermissions os.FileMode
	EnvPrefix         string
	RemoteProvider    []RemoteProvider
	AllowEmptyEnv     bool
	IniLoadOptions    ini.LoadOptions
	EnvVars           []string
}

type RemoteProvider struct {
	Provider      string
	Endpoint      string
	Path          string
	SecretKeyring string
}

// BeforeInject is a no-op satisfying the Config interface.
func (c *Config) BeforeInject() {

}

// AfterInject calls Init after config fields are populated.
func (c *Config) AfterInject() {
	c.Init()
}

// Init sets the default config type to "toml" if not specified.
func (c *Config) Init() *Config {
	if c.ConfigType == "" {
		c.ConfigType = "toml"
	}
	return c
}

// Build creates and configures a new Viper instance (remote or local) based on the config fields.
func (c *Config) Build() (*viper.Viper, error) {
	var runtimeViper = viper.New()
	return runtimeViper, c.build(runtimeViper)
}

// build applies the full configuration (remote/local providers, env, watch) to runtimeViper.
func (c *Config) build(runtimeViper *viper.Viper) error {
	if c.Debug {
		runtimeViper.Debug()
	}
	runtimeViper.SetConfigType(c.ConfigType) // because there is no file extension in a stream of bytes, supported extensions are "json", "toml", "yaml", "yml", "properties", "props", "prop", "Env", "dotenv"
	if len(c.RemoteProvider) > 0 {
		var err error
		for _, conf := range c.RemoteProvider {
			if conf.SecretKeyring != "" {
				err = runtimeViper.AddSecureRemoteProvider(conf.Provider, conf.Endpoint, conf.Path, conf.SecretKeyring)
			} else {
				err = runtimeViper.AddRemoteProvider(conf.Provider, conf.Endpoint, conf.Path)
			}
			if err != nil {
				return err
			}
		}

		// read from remote Config the first time.
		err = runtimeViper.ReadRemoteConfig()
		if err != nil {
			return err
		}
		if c.Watch {
			err = runtimeViper.WatchRemoteConfig()
			if err != nil {
				return err
			}
		}

	} else {

		runtimeViper.SetConfigFile(c.ConfigFile)
		if c.ConfigPermissions > 0 {
			runtimeViper.SetConfigPermissions(c.ConfigPermissions)
		}
		err := runtimeViper.ReadInConfig()
		if err != nil {
			return err
		}
		if c.Watch {
			runtimeViper.WatchConfig()
		}
	}
	runtimeViper.AllowEmptyEnv(c.AllowEmptyEnv)
	runtimeViper.SetEnvPrefix(c.EnvPrefix)
	if len(c.EnvVars) > 0 {
		err := runtimeViper.BindEnv(c.EnvVars...)
		if err != nil {
			return err
		}
	}

	// open a goroutine to watch remote changes forever
	// This remote-watch loop is awkward; left commented for now.
	/*	go func() {
		for {
			time.Sleep(time.Second * 5) // delay after each request

			// currently, only tested with etcd support
			err := runtime_viper.WatchRemoteConfig()
			if err != nil {
				log.Errorf("unable to read remote Config: %v", err)
				continue
			}
			vconf :=runtime_viper.AllSettings()
			log.Debug(vconf)
			// unmarshal new Config into our runtime Config struct. you can also use channel
			// to implement a signal to notify the system of the changes
			runtime_viper.Unmarshal(cCopy)
			refresh(cCopy, dCopy)
			log.Debug(cCopy)
		}
	}()*/
	return nil
}

// Viper is a DaoField wrapper around viper.Viper. Prefer using the global Viper instance instead.
type Viper struct {
	*viper.Viper
	Conf Config
}

// Config returns the embedded Conf as the configuration for injection.
func (v *Viper) Config() any {
	return &v.Conf
}

// Init creates and stores the Viper instance.
func (v *Viper) Init() error {
	var err error
	v.Viper, err = v.Conf.Build()
	return err
}

// Close is a no-op (viper.Viper has no teardown); prefer using the global Viper instance.
func (v *Viper) Close() error {
	return nil
}
