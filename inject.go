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
func (a *App[C, D, CPtr, DPtr]) newStruct(conf Config, dao Dao) any {
	nameValueMap := make(map[string]reflect.Value)
	var structFields []reflect.StructField
	var confValue reflect.Value
	var confType reflect.Type
	// BuiltinConfig participates only in the first injection; hot reloads must
	// not re-merge it. The snapshot is nil until the first inject succeeds, so
	// it doubles as the "first injection" marker.
	if a.confSnapshot.Load() == nil {
		confValue = reflect.ValueOf(&a.BuiltinConfig).Elem()
		confType = confValue.Type()
		for i := range confValue.NumField() {
			field := confValue.Field(i).Addr()

			structField := confType.Field(i)
			name := structField.Name
			tagSettings := fieldTagSettings(structField)
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
					c.BeforeInjectWithRoot(&a.RootConfig)
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
		tagSettings := fieldTagSettings(structField)
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
				c.BeforeInjectWithRoot(&a.RootConfig)
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

					// TODO: strengthen validation — Config() must not return nil
					daoConfig := daoField.Config()
					if daoConfig == nil {
						log.Fatalf("Dao %s Config() return nil", structField.Name)
					}

					name := structField.Name
					daoConfigValue := reflect.ValueOf(daoConfig)
					daoConfigType := reflect.TypeOf(daoConfig)
					tagSettings := fieldTagSettings(structField)
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
						c.BeforeInjectWithRoot(&a.RootConfig)
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
	a.setNewStruct(newStruct.Elem(), nameValueMap)
	return newStruct.Interface()
}

// setNewStruct copies values from typValueMap into the dynamically-created struct fields.
func (a *App[C, D, CPtr, DPtr]) setNewStruct(value reflect.Value, typValueMap map[string]reflect.Value) {
	typ := value.Type()
	for i := range value.NumField() {
		structField := typ.Field(i)
		name := structField.Name
		tagSettings := fieldTagSettings(structField)
		if tagSettings.ConfigName != "" {
			name = stringsx.UpperCaseFirst(tagSettings.ConfigName)
		}

		field := value.Field(i)
		field.Set(typValueMap[name])
	}
}

// inject runs the full lifecycle: config injection followed by dao connection.
// Kept for the extra-inject path (Inject), where the caller passes an additional
// dao that must be connected right away. The initial bootstrap only runs
// injectConfig; the primary dao is connected later via Start.
func (a *App[C, D, CPtr, DPtr]) inject(conf Config, dao Dao) error {
	if err := a.injectConfig(conf, dao); err != nil {
		return err
	}
	if dao == nil {
		return nil
	}
	return a.connectDao(dao)
}

// injectConfig unmarshals the merged config struct from Viper, applies flag overrides,
// and then invokes all AfterInject lifecycle hooks on the config tree and the dao's
// config-only hooks (AfterInjectConfig). It never touches the network: the dao struct
// gets its configuration but no clients are built here. Viper access happens under
// a.mu; user hooks always run outside the lock so they may safely call Defer/Conf
// without deadlocking. Errors are returned to the caller, which decides the policy:
// Start fatals, hot reload and Inject log.
func (a *App[C, D, CPtr, DPtr]) injectConfig(conf Config, dao Dao) error {
	tmpConfig := a.newStruct(conf, dao)
	applyTagDefaults(tmpConfig)
	a.mu.Lock()
	err := a.Viper.Unmarshal(tmpConfig, decoderConfigOptions...)
	if err == nil {
		a.applyFlagConfig(strings.ToLower(a.RootConfig.Name), tmpConfig)
	}
	a.mu.Unlock()
	if err != nil {
		return err
	}
	a.afterInjectConfigCall(tmpConfig)
	conf.AfterInject()
	if c, ok := conf.(afterInjectWithRoot); ok {
		c.AfterInjectWithRoot(&a.RootConfig)
	}
	if dao != nil {
		dao.AfterInjectConfig()
		if c, ok := dao.(afterInjectConfigWithRoot); ok {
			c.AfterInjectConfigWithRoot(&a.RootConfig)
		}
	}
	//log.Debugf("config:  %+v", tmpConfig)
	return nil
}

// connectDao builds the actual clients (GORM/Redis/...) via each DaoField's Init
// and then runs the dao hooks that require live clients (AfterInject, e.g. OTel
// instrumentation). Called by Start for the primary dao and by Inject for extra
// daos. Errors are returned to the caller, which decides the policy: Start
// fatals, Inject logs and returns.
func (a *App[C, D, CPtr, DPtr]) connectDao(dao Dao) error {
	return a.injectDao(dao)
}

// afterInjectConfigCall recursively walks all fields in tmpConfig and calls AfterInject /
// AfterInjectWithRoot on any field that implements the corresponding interface.
func (a *App[C, D, CPtr, DPtr]) afterInjectConfigCall(tmpConfig any) {
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
			a.afterInjectConfigCall(field.Interface())
		}
		if field.CanInterface() {
			inter := field.Interface()
			// Guard against a nil pointer receiver: a typed-nil pointer still
			// satisfies the interface, so dispatching hooks on it would panic.
			if rv := reflect.ValueOf(inter); !(rv.Kind() == reflect.Ptr && rv.IsNil()) {
				if subconf, ok := inter.(afterInject); ok {
					subconf.AfterInject()
				}
				if subconf, ok := inter.(afterInjectWithRoot); ok {
					subconf.AfterInjectWithRoot(&a.RootConfig)
				}
			}
		}
	}
}

// injectDao initializes each DaoField in the Dao struct, skipping entries listed in RootConfig.SkipInjectDaos.
func (a *App[C, D, CPtr, DPtr]) injectDao(dao Dao) error {
	v := reflect.ValueOf(dao).Elem()
	if !v.IsValid() {
		return nil
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
			if slices.Contains(a.RootConfig.SkipInjectDaos, confName) {
				continue
			}

			// Init via DaoField when the field implements it.
			if daofield, ok := inter.(DaoField); ok {
				if err := daofield.Init(); err != nil {
					return err
				}
			}
		}
	}
	dao.AfterInject()
	if c, ok := dao.(afterInjectWithRoot); ok {
		c.AfterInjectWithRoot(&a.RootConfig)
	}
	return nil
}

// Inject injects additional Config/Dao after the app has been started.
// Returns an error if Start has not run yet.
func (a *App[C, D, CPtr, DPtr]) Inject(conf Config, dao Dao) error {
	if !a.started.Load() {
		return errors.New("not started, please call App.Start first")
	}

	if dao != nil {
		a.Defer(func() {
			if err := closeDao(dao); err != nil {
				log.Errorf("close Dao error: %v", err)
			}
		})
	}
	a.reloadMu.Lock()
	defer a.reloadMu.Unlock()
	a.beforeInjectCall(conf, dao)
	return a.inject(conf, dao)
}
