/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"flag"
	"os"
	"reflect"
	"strings"

	kvstruct "github.com/hopeio/gox/kvstruct"
	"github.com/hopeio/gox/log"
	reflectx "github.com/hopeio/gox/reflect"
	stringsx "github.com/hopeio/gox/strings"
	"github.com/hopeio/gox/structtag"
	"github.com/spf13/pflag"
)

// example:
/*
type Dao struct {
	DB mysql.DB `init:"config:MysqlTest"`
}

type Config struct {
	// Environment name.
	Env string `init:"flag:env;short_flag:e;default:dev;usage:environment;env:ENV"`
	// Config file path.
	ConfPath string `init:"flag:config;short_flag:c;usage:config file path;env:CONFIG"`
}
*/

const (
	initTagName = "init"
)

// initTagSettings is the unified per-field tag settings of the "init" tag, parsed
// via gox's meta tags. All segments are flat peers separated by ";":
//
//	config / skip                          config loading (rename, skip injection)
//	flag / short_flag / env / default / usage   command line option & env binding
type initTagSettings struct {
	Skip         bool   `meta:"skip"`
	ConfigName   string `meta:"config"`
	Name         string `meta:"flag"`
	Short        string `meta:"short_flag"`
	Env          string `meta:"env"`
	DefaultValue string `meta:"default"`
	Usage        string `meta:"usage"`
}

// customizesOption reports whether any explicit binding (name/short/env/default/usage)
// is set on the field, i.e. the field opts out of the default env binding derived
// from its field name.
func (s *initTagSettings) customizesOption() bool {
	return s.Name != "" || s.Short != "" || s.Env != "" || s.DefaultValue != "" || s.Usage != ""
}

// fieldTagSettings returns the "init" tag settings of a field. `init:"-"`
// skips the field (autowired or internal), and `init:"skip"` is equivalent.
func fieldTagSettings(field reflect.StructField) *initTagSettings {
	return parseInitTagSettings(field.Tag.Get(initTagName))
}

// parseInitTagSettings parses an "init" struct tag via gox's meta parser. Every
// segment is a flat peer separated by ";": config / skip name the config source or
// skip injection, while flag / short_flag / env / default / usage bind the command
// line flag and environment variable. `flag:NAME` is the flag name (there is no
// `name:` key); `short_flag:S` is the shorthand; a usage value may contain commas
// because segments are ";"-separated.
func parseInitTagSettings(str string) *initTagSettings {
	if str == "" {
		return &initTagSettings{}
	}
	if str == "-" || str == "skip" {
		return &initTagSettings{Skip: true}
	}
	var s initTagSettings
	if err := structtag.ParseSettingTagIntoStruct(str, ";", ":", &s); err != nil {
		log.Fatal(err)
	}
	return &s
}

// envLookupKeys returns env var names to try: ENV uppercased, then NAME_ENV when module name is set.
func envLookupKeys(moduleName, envKey string) []string {
	key := strings.ToUpper(strings.TrimSpace(envKey))
	if key == "" {
		return nil
	}
	keys := []string{key}
	mod := strings.ToUpper(strings.TrimSpace(moduleName))
	if mod != "" && !strings.HasPrefix(key, mod+"_") {
		keys = append(keys, mod+"_"+key)
	}
	return keys
}

func lookupEnvFromKeys(keys []string) (string, bool) {
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			return v, true
		}
	}
	return "", false
}

func bindEnvKeys(v interface{ BindEnv(...string) error }, keys ...string) {
	for _, k := range keys {
		if err := v.BindEnv(k); err != nil {
			log.Fatal(err)
		}
	}
}

func init() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

type anyValue reflect.Value

// String returns the flag's current value as a formatted string.
func (a anyValue) String() string {
	return stringsx.FormatReflectValue(reflect.Value(a))
}

// Type returns the kind name of the underlying reflect.Value (used by pflag).
func (a anyValue) Type() string {
	return reflect.Value(a).Kind().String()
}

// Set parses the string v and assigns it to the underlying reflect.Value.
func (a anyValue) Set(v string) error {
	return kvstruct.ParseStringSetReflectValue(reflect.Value(a), v, nil)
}

// TODO: flag values outrank env and config-center values.

// applyFlagConfig registers and parses all "init" tagged fields in the given config structs,
// then binds the resulting pflag set to Viper.
func (gc *globalConfig[C, D, CPtr, DPtr]) applyFlagConfig(prefix string, confs ...any) {
	commandLine := newCommandLine()
	for _, conf := range confs {
		gc.injectFlagConfig(prefix, commandLine, reflect.ValueOf(conf))
	}

	err := gc.Viper.BindPFlags(commandLine)
	if err != nil {
		log.Fatal(err)
	}

	parseFlag(commandLine)
}

// injectFlagConfig recursively walks a config value, registers each field as a pflag flag,
// and binds default/env values from the "init" tag.
func (gc *globalConfig[C, D, CPtr, DPtr]) injectFlagConfig(prefix string, commandLine *pflag.FlagSet, fcValue reflect.Value) {
	fcValue = reflectx.DerefValue(fcValue)
	if !fcValue.IsValid() {
		return
	}
	fcTyp := fcValue.Type()
	envPrefix := strings.ReplaceAll(strings.ToUpper(prefix), ".", "_")
	for i := range fcTyp.NumField() {
		fieldType := fcTyp.Field(i)
		if !fieldType.IsExported() {
			continue
		}

		fieldValue := fcValue.Field(i)
		kind := fieldValue.Kind()

		if kind == reflect.Pointer || kind == reflect.Interface {
			fieldValue = reflectx.DerefValue(fieldValue)
			kind = fieldValue.Kind()
			if !fieldValue.IsValid() {
				continue
			}
		}

		settings := fieldTagSettings(fieldType)
		// `init:"-"` / `init:"skip"`: autowired or internal field, no env/flag binding.
		if settings.Skip {
			continue
		}

		flag := strings.ToLower(fieldType.Name)
		if prefix != "" {
			flag = prefix + "." + flag
		}
		if kind == reflect.Struct {
			if fieldType.Anonymous {
				gc.injectFlagConfig(prefix, commandLine, fieldValue)
			} else {
				gc.injectFlagConfig(flag, commandLine, fieldValue)
			}
			continue
		}

		if kind == reflect.Chan || kind == reflect.Func || kind == reflect.Interface || kind == reflect.Map ||
			kind == reflect.Struct {
			continue
		}

		if settings.customizesOption() {
			// Tag defaults are applied by applyTagDefaults before decode — not here.
			// From env: try ENV, then MODULE_ENV when RootConfig.Name is set.
			if settings.Env != "" {
				envKeys := envLookupKeys(gc.RootConfig.Name, settings.Env)
				bindEnvKeys(gc.Viper, envKeys...)
				if value, ok := lookupEnvFromKeys(envKeys); ok {
					if err := kvstruct.ParseStringSetReflectValue(fieldValue, value, &fieldType); err != nil {
						log.Fatal(err)
					}
				}
			}
			if !gc.initialized.Load() && settings.Name != "" {
				flagv := &pflag.Flag{
					Name:      settings.Name,
					Shorthand: settings.Short,
					Usage:     settings.Usage,
					Value:     anyValue(fieldValue),
				}
				if kind == reflect.Bool {
					flagv.NoOptDefVal = "true"
				}
				commandLine.AddFlag(flagv)
			}
		} else if kind != reflect.Slice && kind != reflect.Array {
			// default env
			env := strings.ToUpper(fieldType.Name)
			if envPrefix != "" {
				env = envPrefix + "_" + env
			}

			err := gc.Viper.BindEnv(env)
			if err != nil {
				log.Fatal(err)
			}

			if value, ok := os.LookupEnv(env); ok {
				err := kvstruct.ParseStringSetReflectValue(fieldValue, value, &fieldType)
				if err != nil {
					log.Fatal(err)
				}
			}
			// default flag
			if !gc.initialized.Load() {
				flagv := &pflag.Flag{
					Name:  flag[strings.IndexByte(flag, '.')+1:],
					Value: anyValue(fieldValue),
				}
				if kind == reflect.Bool {
					flagv.NoOptDefVal = "true"
				}
				commandLine.AddFlag(flagv)
			}
		}
	}
}

// applyTagDefaults writes each tag's "default" into conf before decode.
//
// Layering is default → config file → env → command line. Running this first
// keeps intentional zeros (Port = 0, Debug = false) meaningful, which a
// post-decode "fill if IsZero" check cannot express. Fields marked
// `init:"-"` are skipped entirely.
func applyTagDefaults(conf any) {
	applyTagDefaultsValue(reflect.ValueOf(conf))
}

func applyTagDefaultsValue(v reflect.Value) {
	v = reflectx.DerefValue(v)
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return
	}
	typ := v.Type()
	for i := range typ.NumField() {
		fieldType := typ.Field(i)
		if !fieldType.IsExported() {
			continue
		}
		settings := fieldTagSettings(fieldType)
		if settings.Skip {
			continue
		}
		fieldValue := v.Field(i)
		if k := fieldValue.Kind(); k == reflect.Pointer || k == reflect.Interface {
			fieldValue = reflectx.DerefValue(fieldValue)
			if !fieldValue.IsValid() {
				continue
			}
		}
		if fieldValue.Kind() == reflect.Struct {
			applyTagDefaultsValue(fieldValue)
			continue
		}
		if settings.DefaultValue == "" {
			continue
		}
		if err := kvstruct.ParseStringSetReflectValue(fieldValue, settings.DefaultValue, &fieldType); err != nil {
			log.Fatal(err)
		}
	}
}

// newCommandLine creates a pflag.FlagSet that silently ignores unknown flags.
func newCommandLine() *pflag.FlagSet {
	commandLine := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	commandLine.ParseErrorsAllowlist.UnknownFlags = true
	return commandLine
}

// parseFlag parses os.Args[1:] into the given flag set, fatal on error.
func parseFlag(commandLine *pflag.FlagSet) {
	err := commandLine.Parse(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
}

// InjectByFlag parses the given args slice and injects flag values into conf.
func (gc *globalConfig[C, D, CPtr, DPtr]) InjectByFlag(args []string, conf any) error {
	applyTagDefaults(conf)
	commandLine := pflag.NewFlagSet(args[0], pflag.ContinueOnError)
	commandLine.ParseErrorsAllowlist.UnknownFlags = true
	gc.injectFlagConfig("", commandLine, reflect.ValueOf(conf))
	return commandLine.Parse(args[1:])
}

// getFieldConfigName returns the effective mapstructure name, tag options, and whether the field should be included.
func getFieldConfigName(v *reflect.StructField) (string, tagOptions, bool) {
	tag := v.Tag.Get("mapstructure")
	if tag == "" {
		return v.Name, "", true
	}
	if tag == "-" {
		return "", "", false
	}
	name, opts := parseTag(tag)
	if name == "" {
		return v.Name, opts, true
	}
	return stringsx.UpperCaseFirst(name), opts, true
}

// tagOptions is the string following a comma in a struct field's "json"
// tag, or the empty string. It does not include the leading comma.
type tagOptions string

// parseTag splits a struct field's json tag into its name and
// comma-separated options.
func parseTag(tag string) (string, tagOptions) {
	tag, opt, _ := strings.Cut(tag, ",")
	return tag, tagOptions(opt)
}

// Contains reports whether a comma-separated list of options
// contains a particular substr flag. substr must be surrounded by a
// string boundary or commas.
func (o tagOptions) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}
	s := string(o)
	for s != "" {
		var name string
		name, s, _ = strings.Cut(s, ",")
		if name == optionName {
			return true
		}
	}
	return false
}
