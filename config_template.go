/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"fmt"
	"os"
	"reflect"

	stringsx "github.com/hopeio/gox/strings"
)

// genConfigTemplate writes a config template file to the directory specified by ConfigTemplateDir.
// In single-file mode the template merges builtin, user config and DAO fields into one file;
// otherwise only the env-local fields are written.
func (a *App[C, D, CPtr, DPtr]) genConfigTemplate(conf CPtr, singleTemplateFileConfig bool) error {
	dir := a.RootConfig.ConfigTemplateDir
	if dir == "" {
		return nil
	}
	if dir[len(dir)-1] != '/' {
		dir += "/"
	}

	format := a.RootConfig.ConfigCenter.Format
	filename := prefixLocalTemplate + format

	confMap := make(map[string]any)
	if singleTemplateFileConfig {
		filename = prefixConfigTemplate + string(format)
		if err := struct2Map(&a.RootConfig.BasicConfig, confMap); err != nil {
			return err
		}
		delete(confMap, fixedFieldNameEnv)
		if err := struct2Map(&a.RootConfig.EnvConfig, confMap); err != nil {
			return err
		}
		delete(confMap, fixedFieldNameConfigCenter)
	}
	if err := struct2Map(&a.BuiltinConfig, confMap); err != nil {
		return err
	}
	if err := struct2Map(conf, confMap); err != nil {
		return err
	}
	if err := daoConfig2Map(reflect.ValueOf(a.Dao).Elem(), confMap); err != nil {
		return err
	}

	encoder, err := codecRegistry.Encoder(format)
	if err != nil {
		return fmt.Errorf("initialize: config template encoder %s: %w", format, err)
	}
	data, err := encoder.Encode(confMap)
	if err != nil {
		return fmt.Errorf("initialize: encode config template: %w", err)
	}
	if err := os.WriteFile(dir+filename, data, 0644); err != nil {
		return fmt.Errorf("initialize: write config template %s: %w", dir+filename, err)
	}
	return nil
}

// daoConfig2Map iterates over DAO fields that implement DaoField and adds their Config() result to confMap.
func daoConfig2Map(value reflect.Value, confMap map[string]any) error {
	typ := value.Type()
	for i := range value.NumField() {
		field := value.Field(i)
		if field.Addr().Type().Implements(DaoFieldType) {
			newconfMap := make(map[string]any)
			fieldType := typ.Field(i)
			name := fieldType.Name
			tagSettings, err := fieldTagSettings(fieldType)
			if err != nil {
				return err
			}
			if tagSettings.ConfigName != "" {
				name = stringsx.UpperCaseFirst(tagSettings.ConfigName)
			}

			confMap[name] = newconfMap
			if err := struct2Map(field.Addr().Interface().(DaoField).Config(), newconfMap); err != nil {
				return err
			}
		}
	}
	return nil
}
