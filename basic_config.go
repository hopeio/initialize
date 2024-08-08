package initialize

import (
	"github.com/hopeio/utils/log"
	stringsi "github.com/hopeio/utils/strings"
	"os"
	"path/filepath"
)

func (gc *globalConfig) setBasicConfig() {
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
	if gc.RootConfig.Module == "" {
		gc.RootConfig.Module = stringsi.CutPart(filepath.Base(os.Args[0]), ".")
	}
	if gc.RootConfig.ConfPath != confPath {
		gc.RootConfig.ConfPath = confPath
	}
}
