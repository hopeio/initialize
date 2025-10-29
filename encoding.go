/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"reflect"
	"slices"

	"github.com/go-viper/mapstructure/v2"
	"github.com/hopeio/gox/encoding"
	stringsx "github.com/hopeio/gox/strings"
	"github.com/spf13/viper"
)

var (
	codecRegistry        = viper.NewCodecRegistry()
	decoderConfigOptions = []viper.DecoderConfigOption{
		viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.TextUnmarshallerHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		)),
		func(config *mapstructure.DecoderConfig) {
			config.Squash = true
		},
	}
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
	structValue2Map(reflect.ValueOf(v).Elem(), nil, confMap)
}

// 递归的根据反射将对象中的指针变量赋值
func structValue2Map(value reflect.Value, field *reflect.StructField, confMap map[string]any) {
	var name string
	var opt tagOptions
	if field != nil {
		// 判断field是否大写
		if !field.IsExported() {
			return
		}
		// 判断是匿名字段
		var ok bool
		name, opt, ok = getFieldConfigName(field)
		if !ok {
			return
		}
		tagSettings := parseInitTagSettings(field.Tag.Get(initTagName))
		if tagSettings.ConfigName != "" {
			name = stringsx.UpperCaseFirst(tagSettings.ConfigName)
		}
	}

	typ := value.Type()
	kind := value.Kind()
	switch kind {
	case reflect.Func, reflect.Chan, reflect.Interface:
		return
	case reflect.Slice, reflect.Array:
		if slices.Contains([]reflect.Kind{reflect.Func, reflect.Chan, reflect.Interface}, typ.Elem().Kind()) {
			return
		}

		if field != nil {
			var values []any
			confMap[name] = values
			if value.Len() > 0 {
				for i := 0; i < value.Len(); i++ {
					values = append(values, value.Index(i).Interface())
				}
			} else {
				newconfMap := make(map[string]any)
				values = append(values, newconfMap)
				confMap[name] = values
				structValue2Map(reflect.New(typ.Elem()).Elem(), nil, newconfMap)
			}
		}
	case reflect.Map:
	case reflect.Ptr:
		typName := typ.Elem().String()
		if slices.Contains(skipInjectTypes, typName) {
			return
		}
		if value.IsNil() {
			value = reflect.New(typ.Elem()).Elem()
		}
		structValue2Map(value, field, confMap)
	case reflect.Struct:
		// 如果是tls.Config 类型，则不处理,这里可能会干扰其他相同的定义
		typName := typ.String()
		if slices.Contains(skipInjectTypes, typName) {
			return
		}

		if field != nil {
			if value.CanSet() {
				if opt == "squash" || field.Anonymous {
					structValue2Map(value, nil, confMap)
				} else {
					newconfMap := make(map[string]any)
					confMap[name] = newconfMap
					structValue2Map(value, nil, newconfMap)
					if len(newconfMap) == 0 {
						delete(confMap, name)
					}
				}
			}
		} else {
			for i := range value.NumField() {
				structField := typ.Field(i)
				structValue2Map(value.Field(i), &structField, confMap)
			}
		}
	default:
		if field != nil && value.CanInterface() {
			confMap[name] = value.Interface()
		}
	}
}
