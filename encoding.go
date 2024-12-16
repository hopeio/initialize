/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"github.com/hopeio/utils/encoding"
	stringsi "github.com/hopeio/utils/strings"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"reflect"
	"slices"
)

type encoder interface {
	Encode(format string, v map[string]any) ([]byte, error)
}

func formatDecoderConfigOption(format encoding.Format) []viper.DecoderConfigOption {
	return append(decoderConfigOptions, func(config *mapstructure.DecoderConfig) {
		if format == encoding.Yml {
			format = encoding.Yaml
		}
		config.TagName = string(format)
	})
}

func struct2Map(v any, confMap map[string]any) {
	structValue2Map(reflect.ValueOf(v).Elem(), confMap)
}

// 递归的根据反射将对象中的指针变量赋值
func structValue2Map(value reflect.Value, confMap map[string]any) {
	typ := value.Type()
	for i := range value.NumField() {
		field := value.Field(i)
		fileKind := field.Kind()
		fieldType := typ.Field(i)
		// 判断field是否大写
		if !fieldType.IsExported() {
			continue
		}
		switch fileKind {
		case reflect.Func, reflect.Chan, reflect.Interface:
			continue
		case reflect.Slice, reflect.Map, reflect.Array:
			if slices.Contains([]reflect.Kind{reflect.Func, reflect.Chan, reflect.Interface}, fieldType.Type.Elem().Kind()) {
				continue
			}
		case reflect.Ptr, reflect.Struct:
			if field.CanSet() {
				// 如果是tls.Config 类型，则不处理,这里可能会干扰其他相同的定义
				typName := fieldType.Type.String()
				if fileKind == reflect.Ptr {
					typName = field.Type().Elem().String()
				}
				if typName == skipTypeTlsConfig {
					continue
				}
				newValue := field
				if fileKind == reflect.Ptr {
					newValue = reflect.New(field.Type().Elem()).Elem()
				}

				// 判断是匿名字段
				name, opt, ok := getFieldConfigName(fieldType)
				if !ok {
					continue
				}
				if opt == "squash" || fieldType.Anonymous {
					structValue2Map(newValue, confMap)
				} else {
					tagSettings := parseInitTagSettings(fieldType.Tag.Get(initTagName))
					if tagSettings.ConfigName != "" {
						name = stringsi.UpperCaseFirst(tagSettings.ConfigName)
					}
					newconfMap := make(map[string]any)
					confMap[name] = newconfMap
					structValue2Map(newValue, newconfMap)
					if len(newconfMap) == 0 {
						delete(confMap, name)
					}
				}
			}
			continue
		}

		if field.CanInterface() {
			name, _, ok := getFieldConfigName(fieldType)
			if !ok {
				continue
			}

			tagSettings := parseInitTagSettings(fieldType.Tag.Get(initTagName))
			if tagSettings.ConfigName != "" {
				name = stringsi.UpperCaseFirst(tagSettings.ConfigName)
			}
			confMap[name] = field.Interface()
		}
	}
}
