/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/hopeio/gox/log"
	"github.com/hopeio/gox/reflect/structtag"
	stringsx "github.com/hopeio/gox/strings"
)

// example:
/*
type Dao struct {
	DB mysql.DB `init:"config:MysqlTest"`
}

type Config struct {
	Env string
}

*/

const (
	initTagName = "init"
	exprTagName = "expr"
)

type initTagSettings struct {
	ConfigName   string `meta:"config"`
	DefaultValue string `meta:"default"`
}

func parseInitTagSettings(str string) *initTagSettings {
	if str == "" {
		return &initTagSettings{}
	}
	var settings initTagSettings
	parseTagSetting(str, &settings)
	return &settings
}

// parseTagSetting default sep ; delimiter :
func parseTagSetting(str string, settings any) {
	err := structtag.ParseSettingTagIntoStruct(str, ";", ":", settings)
	if err != nil {
		log.Fatal(err)
	}
}

func genEncodingTag(name string) string {
	return fmt.Sprintf(`json:"%s" toml:"%s" yaml:"%s"`, name, name, name)
}

// get field name, return filed config name and skip flag
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

// 整合viper,提供单个注入,viper实现也挺简单的,原来的方案就能实现啊
/*func Inject(v Config, path string) {

}
*/
