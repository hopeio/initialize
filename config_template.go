/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"github.com/hopeio/initialize/conf_dao"
	"github.com/hopeio/utils/log"
	stringsi "github.com/hopeio/utils/strings"
	"os"
	"reflect"
	"unsafe"
)

func (gc *globalConfig) genConfigTemplate(singleTemplateFileConfig bool) {
	dir := gc.RootConfig.ConfigTemplateDir
	if dir == "" {
		return
	}
	if dir[len(dir)-1] != '/' {
		dir += "/"
	}

	format := gc.RootConfig.ConfigCenter.Format
	filename := prefixLocalTemplate + string(format)

	confMap := make(map[string]any)
	if singleTemplateFileConfig {
		filename = prefixConfigTemplate + string(format)
		struct2Map(&gc.RootConfig.BasicConfig, confMap)
		delete(confMap, fixedFieldNameEnv)
		struct2Map(&gc.RootConfig.EnvConfig, confMap)
		delete(confMap, fixedFieldNameConfigCenter)
	}
	struct2Map(&gc.BuiltinConfig, confMap)
	struct2Map(gc.conf, confMap)
	if gc.dao != nil {
		daoConfig2Map(reflect.ValueOf(gc.dao).Elem(), confMap)
	}

	encoderRegistry := reflect.ValueOf(gc.Viper).Elem().FieldByName(fixedFieldNameEncoderRegistry).Elem()
	fieldValue := reflect.NewAt(encoderRegistry.Type(), unsafe.Pointer(encoderRegistry.UnsafeAddr()))
	data, err := fieldValue.Interface().(encoder).Encode(format, confMap)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(dir+filename, data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func daoConfig2Map(value reflect.Value, confMap map[string]any) {
	typ := value.Type()
	for i := range value.NumField() {
		field := value.Field(i)
		if field.Addr().Type().Implements(conf_dao.DaoFieldType) {
			newconfMap := make(map[string]any)
			fieldType := typ.Field(i)
			name := fieldType.Name
			tagSettings := parseInitTagSettings(fieldType.Tag.Get(initTagName))
			if tagSettings.ConfigName != "" {
				name = stringsi.UpperCaseFirst(tagSettings.ConfigName)
			}

			confMap[name] = newconfMap
			struct2Map(field.Addr().Interface().(conf_dao.DaoField).Config(), newconfMap)
		}
	}
}
