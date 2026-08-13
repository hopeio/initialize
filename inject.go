/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"errors"
	"reflect"
	"slices"
	"strings"

	"github.com/hopeio/gox/log"
	reflectx "github.com/hopeio/gox/reflect"
	stringsx "github.com/hopeio/gox/strings"
)

// newStruct dynamically builds a merged struct containing all config and DAO config fields,
// calling lifecycle hooks (Init, BeforeInject, BeforeInjectWithRoot) on each sub-config.
func (gc *globalConfig[C, D, CPtr, DPtr]) newStruct(conf Config, dao Dao) any {
	nameValueMap := make(map[string]reflect.Value)
	var structFields []reflect.StructField
	var confValue reflect.Value
	var confType reflect.Type
	// BuiltinConfig
	if !gc.initialized.Load() {
		confValue = reflect.ValueOf(&gc.BuiltinConfig).Elem()
		confType = confValue.Type()
		for i := range confValue.NumField() {
			field := confValue.Field(i).Addr()

			structField := confType.Field(i)
			name := structField.Name
			tagSettings := parseInitTagSettings(structField.Tag.Get(initTagName))
			if tagSettings.ConfigName != "" {
				name = stringsx.UpperCaseFirst(tagSettings.ConfigName)
			}

			if field.CanInterface() {
				inter := field.Interface()
				if c, ok := inter.(Init); ok {
					c.Init()
				}
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
			name = stringsx.UpperCaseFirst(tagSettings.ConfigName)
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

	if dao != nil {
		daoValue := reflect.ValueOf(dao).Elem()
		daoType := daoValue.Type()
		for i := range daoValue.NumField() {
			field := daoValue.Field(i)
			if field.Type().Kind() == reflect.Struct {
				field = field.Addr()
			}
			if field.CanInterface() {
				fieldAny := field.Interface()
				if daoField, ok := fieldAny.(DaoField); ok {

					structField := daoType.Field(i)

					// TODO: 加强校验,必须不为nil
					daoConfig := daoField.Config()
					if daoConfig == nil {
						log.Fatalf("Dao %s Config() return nil", structField.Name)
					}

					name := structField.Name
					daoConfigValue := reflect.ValueOf(daoConfig)
					daoConfigType := reflect.TypeOf(daoConfig)
					tagSettings := parseInitTagSettings(structField.Tag.Get(initTagName))
					if tagSettings.ConfigName != "" {
						name = stringsx.UpperCaseFirst(tagSettings.ConfigName)
					}

					if c, ok := daoConfig.(Init); ok {
						c.Init()
					}
					if c, ok := daoConfig.(beforeInject); ok {
						c.BeforeInject()
					}
					if c, ok := daoConfig.(beforeInjectWithRoot); ok {
						c.BeforeInjectWithRoot(&gc.RootConfig)
					}

					if _, ok := nameValueMap[name]; ok {
						log.Fatalf(`detected %s use same configuration with %s, please rename or use init tag [init:"config:{{otherConfigName}}"]`, structField.Name, name)
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

// setNewStruct copies values from typValueMap into the dynamically-created struct fields.
func (gc *globalConfig[C, D, CPtr, DPtr]) setNewStruct(value reflect.Value, typValueMap map[string]reflect.Value) {
	typ := value.Type()
	for i := range value.NumField() {
		structField := typ.Field(i)
		name := structField.Name
		tagSettings := parseInitTagSettings(structField.Tag.Get(initTagName))
		if tagSettings.ConfigName != "" {
			name = stringsx.UpperCaseFirst(tagSettings.ConfigName)
		}

		field := value.Field(i)
		field.Set(typValueMap[name])
	}
}

// inject unmarshals the merged config struct from Viper, applies flag overrides,
// and then invokes all AfterInject lifecycle hooks on the config and DAO.
// Viper access happens under gc.mu; user hooks always run outside the lock so they
// may safely call Defer/Conf without deadlocking.
func (gc *globalConfig[C, D, CPtr, DPtr]) inject(conf Config, dao Dao) error {
	tmpConfig := gc.newStruct(conf, dao)
	gc.mu.Lock()
	err := gc.Viper.Unmarshal(tmpConfig, decoderConfigOptions...)
	if err == nil {
		gc.applyFlagConfig(strings.ToLower(gc.RootConfig.Name), tmpConfig)
	}
	gc.mu.Unlock()
	if err != nil {
		// 启动期配置错误直接退出；初始化完成后（热更新/追加注入）只记录错误，不能杀死运行中的服务
		if !gc.initialized.Load() {
			log.Fatal(err)
		}
		log.Error(err)
		return err
	}
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
	return nil
}

// afterInjectConfigCall recursively walks all fields in tmpConfig and calls AfterInject /
// AfterInjectWithRoot on any field that implements the corresponding interface.
func (gc *globalConfig[C, D, CPtr, DPtr]) afterInjectConfigCall(tmpConfig any) {
	v := reflectx.DerefValue(reflect.ValueOf(tmpConfig))
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		kind := field.Kind()
		if !v.Type().Field(i).IsExported() {
			continue
		}
		if kind == reflect.Ptr || kind == reflect.Struct || kind == reflect.Interface {
			gc.afterInjectConfigCall(field.Interface())
		}
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

// injectDao initializes each DaoField in the Dao struct, skipping entries listed in RootConfig.SkipInjectDaos.
func (gc *globalConfig[C, D, CPtr, DPtr]) injectDao(dao Dao) {
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
				log.Info("ignore inject pointer type: ", field.Type().String())
				continue
			}
			confName := strings.ToUpper(structFiled.Name)
			if slices.Contains(gc.RootConfig.SkipInjectDaos, confName) {
				continue
			}

			// 根据DaoField接口实现获取配置和要注入的类型
			if daofield, ok := inter.(DaoField); ok {
				err := daofield.Init()
				if err != nil {
					log.Fatal("inject", confName, "err:", err)
				}
			}
		}
	}
	dao.AfterInject()
	if c, ok := dao.(afterInjectWithRoot); ok {
		c.AfterInjectWithRoot(&gc.RootConfig)
	}
}

// Inject injects additional Config/Dao after the global config is already initialized.
// Returns an error if the global config has not yet been initialized.
func (gc *globalConfig[C, D, CPtr, DPtr]) Inject(conf Config, dao Dao) error {
	if !gc.initialized.Load() {
		return errors.New("not initialized, please call initialize.NewGlobal first")
	}

	if dao != nil {
		gc.Defer(func() {
			if err := closeDao(dao); err != nil {
				log.Errorf("close Dao error: %v", err)
			}
		})
	}
	gc.reloadMu.Lock()
	defer gc.reloadMu.Unlock()
	gc.beforeInjectCall(conf, dao)
	return gc.inject(conf, dao)
}
