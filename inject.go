/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"bytes"
	"errors"
	"github.com/hopeio/initialize/conf_dao"
	"github.com/hopeio/utils/log"
	stringsi "github.com/hopeio/utils/strings"
	"reflect"
	"slices"
	"strings"
)

func (gc *globalConfig) UnmarshalAndSet(data []byte) {
	gc.mu.Lock()
	err := gc.Viper.MergeConfig(bytes.NewReader(data))
	if err != nil {
		if gc.editTimes == 0 {
			log.Fatal(err)
		} else {
			log.Error(err)
			return
		}
	}

	gc.inject(gc.conf, gc.dao)
	gc.editTimes++
	gc.mu.Unlock()
}

func (gc *globalConfig) newStruct(conf Config, dao Dao) any {
	nameValueMap := make(map[string]reflect.Value)
	var structFields []reflect.StructField
	var confValue reflect.Value
	var confType reflect.Type
	// BuiltinConfig
	if !gc.initialized {
		confValue = reflect.ValueOf(&gc.BuiltinConfig).Elem()
		confType = confValue.Type()
		for i := range confValue.NumField() {
			field := confValue.Field(i).Addr()

			structField := confType.Field(i)
			name := structField.Name
			tagSettings := parseInitTagSettings(structField.Tag.Get(initTagName))
			if tagSettings.ConfigName != "" {
				name = stringsi.UpperCaseFirst(tagSettings.ConfigName)
			}

			if field.CanInterface() {
				inter := field.Interface()
				if c, ok := inter.(beforeInject); ok {
					c.BeforeInject()
				}
				if c, ok := inter.(beforeInjectWithRoot); ok {
					c.BeforeInjectWithRoot(&gc.RootConfig)
				}
			}
			structFields = append(structFields, reflect.StructField{
				Name:      name,
				Type:      field.Type(),
				Tag:       structField.Tag,
				Anonymous: structField.Anonymous,
			})

			nameValueMap[name] = field
		}
	}
	confValue = reflect.ValueOf(conf).Elem()
	confType = confValue.Type()
	for i := range confValue.NumField() {
		field := confValue.Field(i)
		fieldType := field.Type()
		// panic: reflect: embedded type with methods not implemented if type is not first field // Issue 15924.
		if confValue.Field(i).Type() == EmbeddedPresetsType {
			continue
		}
		if fieldType.Kind() != reflect.Ptr && fieldType.Kind() != reflect.Map {
			field = field.Addr()
		}

		structField := confType.Field(i)
		name := structField.Name
		tagSettings := parseInitTagSettings(structField.Tag.Get(initTagName))
		if tagSettings.ConfigName != "" {
			name = stringsi.UpperCaseFirst(tagSettings.ConfigName)
		}

		if v, ok := nameValueMap[name]; ok {
			if v.Type() == field.Type() {
				log.Fatalf(`exists builtin config field: %s, please delete the field`, name)
			} else {
				log.Fatalf(`exists builtin config field: %s, please rename or use init tag [init:"config:{{other config name}}"]`, name)
			}
		}

		if field.CanInterface() {
			inter := field.Interface()
			if c, ok := inter.(beforeInject); ok {
				c.BeforeInject()
			}
			if c, ok := inter.(beforeInjectWithRoot); ok {
				c.BeforeInjectWithRoot(&gc.RootConfig)
			}
		}

		structFields = append(structFields, reflect.StructField{
			Name:      name,
			Type:      field.Type(),
			Tag:       structField.Tag,
			Anonymous: structField.Anonymous,
		})

		nameValueMap[name] = field
	}
	// 不进行二次注入,无法确定业务中是否仍然使用,除非每次加锁,或者说每次业务中都交给一个零时变量?需要规范去控制
	if dao != nil {
		daoValue := reflect.ValueOf(dao).Elem()
		daoType := daoValue.Type()
		for i := range daoValue.NumField() {
			field := daoValue.Field(i)
			if field.Type().Kind() == reflect.Struct {
				field = field.Addr()
			}
			if field.CanInterface() {
				inter := field.Interface()
				if daoField, ok := inter.(conf_dao.DaoField); ok {

					structField := daoType.Field(i)

					// TODO: 加强校验,必须不为nil
					daoConfig := daoField.Config()
					if daoConfig == nil {
						log.Fatalf("dao %s Config() return nil", structField.Name)
					}

					name := structField.Name
					daoConfigValue := reflect.ValueOf(daoConfig)
					daoConfigType := reflect.TypeOf(daoConfig)
					tagSettings := parseInitTagSettings(structField.Tag.Get(initTagName))
					if tagSettings.ConfigName != "" {
						name = stringsi.UpperCaseFirst(tagSettings.ConfigName)
					}

					if c, ok := daoConfig.(beforeInject); ok {
						c.BeforeInject()
					}
					if c, ok := inter.(beforeInjectWithRoot); ok {
						c.BeforeInjectWithRoot(&gc.RootConfig)
					}

					if _, ok := nameValueMap[name]; ok {
						log.Fatalf(`exists field: %s, please rename or use init tag [init:"{{otherConfigName}}"]`, name)
					}

					structFields = append(structFields, reflect.StructField{
						Name: name,
						Type: daoConfigType,
						Tag:  structField.Tag,
					})
					nameValueMap[name] = daoConfigValue
				}
			}
		}
	}
	typ := reflect.StructOf(structFields)
	newStruct := reflect.New(typ)
	gc.setNewStruct(newStruct.Elem(), nameValueMap)
	return newStruct.Interface()
}

func (gc *globalConfig) setNewStruct(value reflect.Value, typValueMap map[string]reflect.Value) {
	typ := value.Type()
	for i := range value.NumField() {
		structField := typ.Field(i)
		name := structField.Name
		tagSettings := parseInitTagSettings(structField.Tag.Get(initTagName))
		if tagSettings.ConfigName != "" {
			name = stringsi.UpperCaseFirst(tagSettings.ConfigName)
		}

		field := value.Field(i)
		field.Set(typValueMap[name])
	}
}

// 注入配置及生成DAO
func (gc *globalConfig) inject(conf Config, dao Dao) {
	tmpConfig := gc.newStruct(conf, dao)
	err := gc.Viper.Unmarshal(tmpConfig, decoderConfigOptions...)
	if err != nil {
		if gc.editTimes == 0 {
			log.Fatal(err)
		} else {
			log.Error(err)
			return
		}
	}
	applyFlagConfig(gc.Viper, tmpConfig)
	gc.afterInjectConfigCall(tmpConfig)
	conf.AfterInject()
	if c, ok := conf.(afterInjectWithRoot); ok {
		c.AfterInjectWithRoot(&gc.RootConfig)
	}
	if dao != nil {
		dao.AfterInjectConfig()
		if c, ok := dao.(afterInjectConfigWithRoot); ok {
			c.AfterInjectConfigWithRoot(&gc.RootConfig)
		}
		gc.injectDao(dao)
	}
	//log.Debugf("config:  %+v", tmpConfig)
}

func (gc *globalConfig) afterInjectConfigCall(tmpConfig any) {
	v := reflect.ValueOf(tmpConfig).Elem()
	if !v.IsValid() {
		return
	}
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.CanInterface() {
			inter := field.Interface()
			if subconf, ok := inter.(afterInject); ok {
				subconf.AfterInject()
			}
			if subconf, ok := inter.(afterInjectWithRoot); ok {
				subconf.AfterInjectWithRoot(&gc.RootConfig)
			}
		}
	}
}

func (gc *globalConfig) injectDao(dao Dao) {
	v := reflect.ValueOf(dao).Elem()
	if !v.IsValid() {
		return
	}
	typ := v.Type()

	for i := range v.NumField() {
		field := v.Field(i)
		structFiled := typ.Field(i)
		if field.Addr().CanInterface() {
			inter := field.Addr().Interface()

			if field.Kind() != reflect.Struct {
				log.Debug("ignore inject pointer type: ", field.Type().String())
				continue
			}
			confName := strings.ToUpper(structFiled.Name)
			if slices.Contains(gConfig.RootConfig.NoInject, confName) {
				continue
			}

			// 根据DaoField接口实现获取配置和要注入的类型
			if daofield, ok := inter.(conf_dao.DaoField); ok {
				err := daofield.Init()
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
	dao.AfterInject()
	if c, ok := dao.(afterInjectWithRoot); ok {
		c.AfterInjectWithRoot(&gc.RootConfig)
	}
}

// 当初始化完成后,仍然有需要注入的config和dao
func (gc *globalConfig) Inject(conf Config, dao Dao) error {
	if !gc.initialized {
		return errors.New("not initialize, please call initialize.initHandler or initialize.Start")
	}
	gc.setConfDao(conf, dao)
	gc.beforeInjectCall(conf, dao)
	gc.inject(conf, dao)
	return nil
}
