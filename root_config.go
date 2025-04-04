/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"github.com/hopeio/utils/log"
	stringsi "github.com/hopeio/utils/strings"
	"os"
	"path/filepath"
)

func (gc *globalConfig[C, D]) setRootConfig() {
	format := gc.RootConfig.ConfigCenter.Format
	confPath := gc.RootConfig.ConfPath

	err := gc.Viper.Unmarshal(&gc.RootConfig, decoderConfigOptions...)
	if err != nil {
		log.Fatal(err)
	}
	applyFlagConfig(nil, &gc.RootConfig)
	if gc.RootConfig.ConfigCenter.Format == "" {
		gc.RootConfig.ConfigCenter.Format = format
	}
	if gc.RootConfig.Name == "" {
		gc.RootConfig.Name = stringsi.CutPart(filepath.Base(os.Args[0]), ".")
	}
	if gc.RootConfig.ConfPath != confPath {
		gc.RootConfig.ConfPath = confPath
	}
}
