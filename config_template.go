/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"os"
	"reflect"

	"github.com/hopeio/gox/log"
	stringsx "github.com/hopeio/gox/strings"
)

// genConfigTemplate writes a config template file to the directory specified by ConfigTemplateDir.
// In single-file mode the template merges builtin, user config and DAO fields into one file;
// otherwise only the env-local fields are written.
func (gc *globalConfig[C, D]) genConfigTemplate(singleTemplateFileConfig bool) {
	dir := gc.RootConfig.ConfigTemplateDir
	if dir == "" {
		return
	}
	if dir[len(dir)-1] != '/' {
		dir += "/"
	}

	format := gc.RootConfig.ConfigCenter.Format
	filename := prefixLocalTemplate + format

	confMap := make(map[string]any)
	if singleTemplateFileConfig {
		filename = prefixConfigTemplate + string(format)
		struct2Map(&gc.RootConfig.BasicConfig, confMap)
		delete(confMap, fixedFieldNameEnv)
		struct2Map(&gc.RootConfig.EnvConfig, confMap)
		delete(confMap, fixedFieldNameConfigCenter)
	}
	struct2Map(&gc.BuiltinConfig, confMap)
	struct2Map(gc.Config, confMap)
	daoConfig2Map(reflect.ValueOf(gc.Dao).Elem(), confMap)

	endocer, err := codecRegistry.Encoder(format)
	if err != nil {
		log.Fatal(err)
	}
	data, err := endocer.Encode(confMap)
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(dir+filename, data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

// daoConfig2Map iterates over DAO fields that implement DaoField and adds their Config() result to confMap.
func daoConfig2Map(value reflect.Value, confMap map[string]any) {
	typ := value.Type()
	for i := range value.NumField() {
		field := value.Field(i)
		if field.Addr().Type().Implements(DaoFieldType) {
			newconfMap := make(map[string]any)
			fieldType := typ.Field(i)
			name := fieldType.Name
			tagSettings := parseInitTagSettings(fieldType.Tag.Get(initTagName))
			if tagSettings.ConfigName != "" {
				name = stringsx.UpperCaseFirst(tagSettings.ConfigName)
			}

			confMap[name] = newconfMap
			struct2Map(field.Addr().Interface().(DaoField).Config(), newconfMap)
		}
	}
}
