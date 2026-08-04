/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */
package initialize

import (
	"reflect"
	"testing"
)

type initRenameConfig struct {
	EmbeddedPresets
	Value string `init:"config:foo"`
}

func TestParseInitTagSettings_BaseCases(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s := parseInitTagSettings("")
		if s.Skip || s.ConfigName != "" || s.DefaultValue != "" {
			t.Fatalf("unexpected settings: %+v", s)
		}
	})

	t.Run("dash", func(t *testing.T) {
		s := parseInitTagSettings("-")
		if !s.Skip || s.ConfigName != "" || s.DefaultValue != "" {
			t.Fatalf("unexpected settings: %+v", s)
		}
	})
}

func TestParseInitTagSettings_ValidTags(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		s := parseInitTagSettings("skip")
		if !s.Skip || s.ConfigName != "" || s.DefaultValue != "" {
			t.Fatalf("unexpected settings: %+v", s)
		}
	})

	t.Run("config only", func(t *testing.T) {
		s := parseInitTagSettings("config:MyConf")
		if s.Skip || s.ConfigName != "MyConf" || s.DefaultValue != "" {
			t.Fatalf("unexpected settings: %+v", s)
		}
	})

	t.Run("default only", func(t *testing.T) {
		s := parseInitTagSettings("default:xyz")
		if s.Skip || s.ConfigName != "" || s.DefaultValue != "xyz" {
			t.Fatalf("unexpected settings: %+v", s)
		}
	})

	t.Run("all", func(t *testing.T) {
		s := parseInitTagSettings("skip;config:MyConf;default:xyz")
		if !s.Skip || s.ConfigName != "MyConf" || s.DefaultValue != "xyz" {
			t.Fatalf("unexpected settings: %+v", s)
		}
	})
}

func TestParseTagAndContains(t *testing.T) {
	name, opts := parseTag("abc,foo,bar")
	if name != "abc" {
		t.Fatalf("name mismatch: got=%q want=%q", name, "abc")
	}
	if !opts.Contains("foo") {
		t.Fatalf("expected contains foo, opts=%q", string(opts))
	}
	if opts.Contains("baz") {
		t.Fatalf("did not expect contains baz, opts=%q", string(opts))
	}
	if opts.Contains("") {
		t.Fatalf("did not expect contains empty option, opts=%q", string(opts))
	}

	name2, opts2 := parseTag("abc")
	if name2 != "abc" || string(opts2) != "" {
		t.Fatalf("unexpected parse result: name=%q opts=%q", name2, string(opts2))
	}
}

func TestGetFieldConfigName(t *testing.T) {
	type demo struct {
		A int
		B int `mapstructure:"b"`
		C int `mapstructure:"c,skip"`
		D int `mapstructure:"-"`
	}

	typ := reflect.TypeOf(demo{})

	t.Run("no mapstructure tag", func(t *testing.T) {
		f, _ := typ.FieldByName("A")
		gotName, gotOpts, ok := getFieldConfigName(&f)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if gotName != "A" {
			t.Fatalf("name mismatch: got=%q want=%q", gotName, "A")
		}
		if gotOpts != "" {
			t.Fatalf("opts mismatch: got=%q want=%q", string(gotOpts), "")
		}
	})

	t.Run("mapstructure name only", func(t *testing.T) {
		f, _ := typ.FieldByName("B")
		gotName, gotOpts, ok := getFieldConfigName(&f)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if gotName != "B" { // UpperCaseFirst("b") == "B"
			t.Fatalf("name mismatch: got=%q want=%q", gotName, "B")
		}
		if string(gotOpts) != "" {
			t.Fatalf("opts mismatch: got=%q want=%q", string(gotOpts), "")
		}
	})

	t.Run("mapstructure name with option", func(t *testing.T) {
		f, _ := typ.FieldByName("C")
		gotName, gotOpts, ok := getFieldConfigName(&f)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if gotName != "C" { // UpperCaseFirst("c") == "C"
			t.Fatalf("name mismatch: got=%q want=%q", gotName, "C")
		}
		if !gotOpts.Contains("skip") {
			t.Fatalf("expected opts to contain skip, opts=%q", string(gotOpts))
		}
	})

	t.Run("mapstructure skip field", func(t *testing.T) {
		f, _ := typ.FieldByName("D")
		gotName, gotOpts, ok := getFieldConfigName(&f)
		if ok {
			t.Fatal("expected ok=false")
		}
		if gotName != "" || gotOpts != "" {
			t.Fatalf("unexpected result: name=%q opts=%q", gotName, string(gotOpts))
		}
	})
}

func TestNewStruct_InitTagConfigName_RenameField(t *testing.T) {
	// 只测 init tag 对“临时注入 struct 字段名/指针绑定”的影响，
	// 不走 viper 注入，避免对配置文件/环境敏感。
	gc := newGlobal[*initRenameConfig, *EmbeddedPresets]()
	conf := &initRenameConfig{}

	tmp := gc.newStruct(conf, nil)
	tmpV := reflect.ValueOf(tmp)
	if tmpV.Kind() != reflect.Ptr || tmpV.Elem().Kind() != reflect.Struct {
		t.Fatalf("unexpected tmp value: %T", tmp)
	}

	tmpElem := tmpV.Elem()
	if _, ok := tmpElem.Type().FieldByName("Value"); ok {
		t.Fatalf("init tag should rename field; unexpected field %q exists", "Value")
	}
	f, ok := tmpElem.Type().FieldByName("Foo")
	if !ok {
		t.Fatalf("expected renamed field %q not found; fields=%v", "Foo", listStructFieldNames(tmpElem.Type()))
	}
	if f.Type.Kind() != reflect.Pointer {
		t.Fatalf("expected %q to be a pointer field, got=%v", "Foo", f.Type)
	}

	ptr := tmpElem.FieldByName("Foo").Interface().(*string)
	if ptr != &conf.Value {
		t.Fatalf("expected tmp field to point to original field; got=%p want=%p", ptr, &conf.Value)
	}
}

func listStructFieldNames(t reflect.Type) []string {
	names := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		names = append(names, t.Field(i).Name)
	}
	return names
}

