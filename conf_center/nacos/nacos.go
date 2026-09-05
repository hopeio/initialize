/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package nacos

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/hopeio/initialize/contrib/nacos"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/file"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

var ConfigCenter = &Nacos{}

type Nacos struct {
	Conf   Config
	Client config_client.IConfigClient
}

type Config struct {
	nacos.Config
	ConfigParams []vo.ConfigParam
}

// Type returns the identifier string "nacos" for this config center.
func (cc *Nacos) Type() string {
	return "nacos"
}

// Config returns the embedded Conf as the configuration for injection.
func (cc *Nacos) Config() any {
	return &cc.Conf
}

// Handle connects to Nacos, fetches each ConfigParam, pre-populates the listen cache, and registers change listeners.
func (cc *Nacos) Handle(ctx context.Context, merge func(io.Reader) error, onChange func(io.Reader) error) error {
	if cc.Client == nil {
		var err error
		cc.Client, err = cc.Conf.Config.Build()
		if err != nil {
			return err
		}
	}
	for _, configParam := range cc.Conf.ConfigParams {
		config, err := cc.Client.GetConfig(configParam)
		if err != nil {
			return err
		}
		// nacos-go-sdk quirk: the first GetConfig cache lives under cache/, while
		// Listen caches under cache/config and is async. To sync once without
		// firing OnChange when nothing changed, write into the listen cache dir
		// (extra read/write cost).
		cacheDir := file.GetCurrentPath() + string(os.PathSeparator) + "cache/config"
		cacheKey := util.GetConfigCacheKey(configParam.DataId, configParam.Group, cc.Conf.ClientConfig.NamespaceId)
		cache.WriteConfigToFile(cacheKey, cacheDir, config)
		merge(strings.NewReader(config))
		configParam.OnChange = func(namespace, group, dataId, data string) {
			onChange(strings.NewReader(data))
		}

		err = cc.Client.ListenConfig(configParam)
		if err != nil {
			return err
		}
	}
	return nil
}

// Close closes the Nacos config client if it was initialized.
func (cc *Nacos) Close() error {
	if cc.Client != nil {
		cc.Client.CloseClient()
	}
	return nil
}
