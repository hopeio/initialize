/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"flag"
	"github.com/hopeio/utils/reflect/mtos"
	"github.com/spf13/viper"

	"github.com/hopeio/utils/log"
	reflecti "github.com/hopeio/utils/reflect"
	"github.com/hopeio/utils/reflect/converter"
	"github.com/spf13/pflag"
	"os"
	"reflect"
)

const flagTagName = "flag"

// TODO: 优先级高于其他Config,覆盖环境变量及配置中心的配置
// example
/*type FlagConfig struct {
	// environment
	Env string `flag:"name:env;short:e;default:dev;usage:环境"`
	// 配置文件路径
	ConfPath string `flag:"name:confdao;short:c;default:config.toml;usage:配置文件路径,默认./config.toml或./config/config.toml"`
}*/

type flagTagSettings struct {
	Name    string `meta:"name"`
	Short   string `meta:"short"`
	Env     string `meta:"env" explain:"从环境变量读取"`
	Default string `meta:"default"`
	Usage   string `meta:"usage"`
}

func init() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// TODO: 绑定flag会在Unmarshal覆盖同名配置,parseFlag会纠正,但设计似乎不太合理,还是有不一致的情况
	applyFlagConfig(gConfig.Viper, &gConfig.RootConfig)
	gConfig.RootConfig.AfterInject()
}

type anyValue reflect.Value

func (a anyValue) String() string {
	return converter.String(reflect.Value(a))
}

func (a anyValue) Type() string {
	return reflect.Value(a).Kind().String()
}

func (a anyValue) Set(v string) error {
	return mtos.SetValueByString(reflect.Value(a), v)
}

func injectFlagConfig(commandLine *pflag.FlagSet, viper *viper.Viper, fcValue reflect.Value) {
	fcValue = reflecti.DerefValue(fcValue)
	if !fcValue.IsValid() {
		return
	}
	fcTyp := fcValue.Type()

	for i := range fcTyp.NumField() {
		fieldType := fcTyp.Field(i)
		if !fieldType.IsExported() {
			continue
		}
		flagTag := fieldType.Tag.Get(flagTagName)
		fieldValue := fcValue.Field(i)
		kind := fieldValue.Kind()
		if kind == reflect.Pointer || kind == reflect.Interface {
			fieldValue = reflecti.DerefValue(fieldValue)
			if fieldValue.Kind() == reflect.Pointer && fieldValue.IsNil() {
				continue
			}
			kind = fieldValue.Kind()
		}
		if flagTag != "" {
			var flagTagSettings flagTagSettings
			parseTagSetting(flagTag, ";", &flagTagSettings)
			// 从环境变量设置
			if flagTagSettings.Env != "" {
				if viper != nil {
					err := viper.BindEnv(flagTagSettings.Env)
					if err != nil {
						log.Fatal(err)
					}
				}
				if value, ok := os.LookupEnv(flagTagSettings.Env); ok {
					err := mtos.SetValueByString(fcValue.Field(i), value)
					if err != nil {
						log.Fatal(err)
					}
				}
			}
			if flagTagSettings.Name != "" {
				// flag设置
				flag := commandLine.VarPF(anyValue(fieldValue), flagTagSettings.Name, flagTagSettings.Short, flagTagSettings.Usage)
				if kind == reflect.Bool {
					flag.NoOptDefVal = "true"
				}
			}
		} else if kind == reflect.Struct {
			injectFlagConfig(commandLine, viper, fieldValue)
		}
	}
}

func applyFlagConfig(viper *viper.Viper, confs ...any) {
	commandLine := newCommandLine()
	for _, conf := range confs {
		fcValue := reflect.ValueOf(conf).Elem()
		injectFlagConfig(commandLine, viper, fcValue)
	}
	if viper != nil {
		err := viper.BindPFlags(commandLine)
		if err != nil {
			log.Fatal(err)
		}
	}
	parseFlag(commandLine)
}

func newCommandLine() *pflag.FlagSet {
	commandLine := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	commandLine.ParseErrorsWhitelist.UnknownFlags = true
	return commandLine
}

func parseFlag(commandLine *pflag.FlagSet) {
	err := commandLine.Parse(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
}
