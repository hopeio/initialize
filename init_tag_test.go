/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */
package initialize

import (
	"reflect"
	"testing"
	"time"
)

type initRenameConfig struct {
	EmbeddedPresets
	Value string `init:"config:foo"`
}

func TestParseInitTagSettings_BaseCases(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s := parseInitTagSettings("")
		if s.Skip || s.ConfigName != "" || s.customizesOption() {
			t.Fatalf("unexpected settings: %+v", s)
		}
	})

	t.Run("dash", func(t *testing.T) {
		s := parseInitTagSettings("-")
		if !s.Skip || s.ConfigName != "" || s.customizesOption() {
			t.Fatalf("unexpected settings: %+v", s)
		}
	})
}

func TestParseInitTagSettings_ValidTags(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		s := parseInitTagSettings("skip")
		if !s.Skip || s.ConfigName != "" || s.customizesOption() {
			t.Fatalf("unexpected settings: %+v", s)
		}
	})

	t.Run("config only", func(t *testing.T) {
		s := parseInitTagSettings("config:MyConf")
		if s.Skip || s.ConfigName != "MyConf" || s.customizesOption() {
			t.Fatalf("unexpected settings: %+v", s)
		}
	})

	t.Run("flag only", func(t *testing.T) {
		s := parseInitTagSettings("flag:env;short_flag:e;default:dev;usage:environment;env:ENV")
		if s.Skip || s.ConfigName != "" {
			t.Fatalf("unexpected settings: %+v", s)
		}
		f := s
		if f.Flag != "env" || f.ShortFlag != "e" || f.Default != "dev" || f.Usage != "environment" || f.Env != "ENV" {
			t.Fatalf("unexpected flag settings: %+v", f)
		}
	})

	t.Run("flag env only", func(t *testing.T) {
		s := parseInitTagSettings("env:MINIO_ACCESS_KEY")
		if s.Flag != "" || s.Env != "MINIO_ACCESS_KEY" {
			t.Fatalf("unexpected flag settings: %+v", s)
		}
	})

	t.Run("all", func(t *testing.T) {
		s := parseInitTagSettings("skip;config:MyConf;flag:env;default:xyz")
		if !s.Skip || s.ConfigName != "MyConf" || s.Default != "xyz" {
			t.Fatalf("unexpected settings: %+v", s)
		}
	})

	t.Run("default only does not customize", func(t *testing.T) {
		s := parseInitTagSettings("default:8080")
		if s.customizesOption() || s.Default != "8080" {
			t.Fatalf("lone default must not customizeOption: %+v", s)
		}
	})
}

func TestParseFlagSegment_UsageMayContainCommas(t *testing.T) {
	s := parseInitTagSettings("flag:config;short_flag:c;usage:config file path, default ./config.xxx;env:CONFIG")
	if s.Flag != "config" || s.ShortFlag != "c" {
		t.Fatalf("unexpected settings: %+v", s)
	}
	if s.Usage != "config file path, default ./config.xxx" {
		t.Fatalf("Usage=%q", s.Usage)
	}
	if s.Env != "CONFIG" {
		t.Fatalf("Env=%q", s.Env)
	}
}

func TestFieldTagSettings_InitTagParsesFlagKeys(t *testing.T) {
	type demo struct {
		Name string `init:"flag:name;short_flag:n;default:def;usage:name;env:NAME_TAG"`
	}
	f, _ := reflect.TypeOf(demo{}).FieldByName("Name")
	s := fieldTagSettings(f)
	if s.Flag != "name" || s.ShortFlag != "n" || s.Default != "def" || s.Env != "NAME_TAG" {
		t.Fatalf("unexpected settings: %+v", s)
	}
}

func TestFieldTagSettings_InitDashSkips(t *testing.T) {
	type demo struct {
		Auto string `init:"-"`
	}
	f, _ := reflect.TypeOf(demo{}).FieldByName("Auto")
	if s := fieldTagSettings(f); !s.Skip {
		t.Fatalf("init:\"-\" should set Skip, got %+v", s)
	}
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
	// Only exercise init-tag rename / pointer wiring on the temp inject struct;
	// skip Viper so the test is not sensitive to config files or the environment.
	gc := newGlobal[initRenameConfig, EmbeddedPresets]()
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

type injectFlagDefaultEnv struct {
	Port  int
	Debug bool
}

type injectFlagNested struct {
	Server injectFlagDefaultEnv
}

type injectFlagEnvTag struct {
	Secret string `init:"env:TOKEN_SECRET"`
}

// Equivalent semantics via a unified init tag (alternative spelling).
type injectInitTagged struct {
	Name string        `init:"flag:name;short_flag:n;default:def;usage:name;env:NAME_TAG"`
	Age  int           `init:"flag:age;default:18;usage:age"`
	Wait time.Duration `init:"flag:wait;default:1s;usage:wait"`
	Auto string        `init:"-"`
}

func TestEnvLookupKeys(t *testing.T) {
	if got := envLookupKeys("hoper", "TOKEN_SECRET"); len(got) != 2 || got[0] != "TOKEN_SECRET" || got[1] != "HOPER_TOKEN_SECRET" {
		t.Fatalf("envLookupKeys(hoper, TOKEN_SECRET)=%v", got)
	}
	if got := envLookupKeys("hoper", "HOPER_TOKEN_SECRET"); len(got) != 1 || got[0] != "HOPER_TOKEN_SECRET" {
		t.Fatalf("already prefixed=%v", got)
	}
	if got := envLookupKeys("", "TOKEN_SECRET"); len(got) != 1 || got[0] != "TOKEN_SECRET" {
		t.Fatalf("no module=%v", got)
	}
}

func TestInjectByFlag_EnvTagDirect(t *testing.T) {
	t.Setenv("TOKEN_SECRET", "direct")
	gc := newGlobal[UserConfig, EmbeddedPresets]()
	gc.RootConfig.Name = "hoper"
	conf := &injectFlagEnvTag{}
	if err := gc.InjectByFlag([]string{"prog"}, conf); err != nil {
		t.Fatal(err)
	}
	if conf.Secret != "direct" {
		t.Fatalf("Secret=%q want direct", conf.Secret)
	}
}

func TestInjectByFlag_EnvTagModulePrefixed(t *testing.T) {
	t.Setenv("HOPER_TOKEN_SECRET", "prefixed")
	gc := newGlobal[UserConfig, EmbeddedPresets]()
	gc.RootConfig.Name = "hoper"
	conf := &injectFlagEnvTag{}
	if err := gc.InjectByFlag([]string{"prog"}, conf); err != nil {
		t.Fatal(err)
	}
	if conf.Secret != "prefixed" {
		t.Fatalf("Secret=%q want prefixed", conf.Secret)
	}
}

func TestInjectByFlag_EnvTagDirectWinsOverPrefixed(t *testing.T) {
	t.Setenv("TOKEN_SECRET", "direct")
	t.Setenv("HOPER_TOKEN_SECRET", "prefixed")
	gc := newGlobal[UserConfig, EmbeddedPresets]()
	gc.RootConfig.Name = "hoper"
	conf := &injectFlagEnvTag{}
	if err := gc.InjectByFlag([]string{"prog"}, conf); err != nil {
		t.Fatal(err)
	}
	if conf.Secret != "direct" {
		t.Fatalf("Secret=%q want direct (ENV before NAME_ENV)", conf.Secret)
	}
}

func TestInjectByFlag_OverridesDefaults(t *testing.T) {
	gc := newGlobal[UserConfig, EmbeddedPresets]()
	conf := &injectInitTagged{}
	err := gc.InjectByFlag([]string{"prog", "--name", "cli", "--age", "30", "--wait", "2h"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	if conf.Name != "cli" {
		t.Fatalf("Name=%q want cli", conf.Name)
	}
	if conf.Age != 30 {
		t.Fatalf("Age=%d want 30", conf.Age)
	}
	if conf.Wait != 2*time.Hour {
		t.Fatalf("Wait=%v want 2h", conf.Wait)
	}
}

func TestInjectByFlag_AppliesTagDefaults(t *testing.T) {
	gc := newGlobal[UserConfig, EmbeddedPresets]()
	conf := &injectInitTagged{}
	err := gc.InjectByFlag([]string{"prog"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	if conf.Name != "def" {
		t.Fatalf("Name=%q want def", conf.Name)
	}
	if conf.Age != 18 {
		t.Fatalf("Age=%d want 18", conf.Age)
	}
	if conf.Wait != time.Second {
		t.Fatalf("Wait=%v want 1s", conf.Wait)
	}
}

func TestInjectByFlag_DefaultEnvWithoutOption(t *testing.T) {
	t.Setenv("PORT", "9090")
	gc := newGlobal[UserConfig, EmbeddedPresets]()
	conf := &injectFlagDefaultEnv{}
	err := gc.InjectByFlag([]string{"prog"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	if conf.Port != 9090 {
		t.Fatalf("Port=%d want 9090", conf.Port)
	}
}

func TestInjectByFlag_NestedStructEnvPrefix(t *testing.T) {
	t.Setenv("SERVER_PORT", "7070")
	gc := newGlobal[UserConfig, EmbeddedPresets]()
	conf := &injectFlagNested{}
	err := gc.InjectByFlag([]string{"prog"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	if conf.Server.Port != 7070 {
		t.Fatalf("Server.Port=%d want 7070", conf.Server.Port)
	}
}

func TestInjectByFlag_BoolNoOptDefVal(t *testing.T) {
	gc := newGlobal[UserConfig, EmbeddedPresets]()
	conf := &injectFlagDefaultEnv{}
	err := gc.InjectByFlag([]string{"prog", "--debug"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	if !conf.Debug {
		t.Fatal("Debug want true with --debug")
	}

	conf2 := &injectFlagDefaultEnv{}
	err = gc.InjectByFlag([]string{"prog", "--debug=false"}, conf2)
	if err != nil {
		t.Fatal(err)
	}
	if conf2.Debug {
		t.Fatal("Debug want false with --debug=false")
	}
}

func TestInjectByFlag_CLIOverridesEnv(t *testing.T) {
	t.Setenv("PORT", "1111")
	gc := newGlobal[UserConfig, EmbeddedPresets]()
	conf := &injectFlagDefaultEnv{}
	err := gc.InjectByFlag([]string{"prog", "--port", "2222"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	if conf.Port != 2222 {
		t.Fatalf("Port=%d want 2222 (CLI over env)", conf.Port)
	}
}

func TestInjectByFlag_InitTagDefaults(t *testing.T) {
	gc := newGlobal[UserConfig, EmbeddedPresets]()
	conf := &injectInitTagged{}
	if err := gc.InjectByFlag([]string{"prog"}, conf); err != nil {
		t.Fatal(err)
	}
	if conf.Name != "def" || conf.Age != 18 || conf.Wait != time.Second {
		t.Fatalf("init tag defaults not applied: %+v", conf)
	}
}

func TestInjectByFlag_InitTagCLI(t *testing.T) {
	gc := newGlobal[UserConfig, EmbeddedPresets]()
	conf := &injectInitTagged{}
	err := gc.InjectByFlag([]string{"prog", "--name", "cli", "--age", "30"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	if conf.Name != "cli" || conf.Age != 30 {
		t.Fatalf("Name=%q Age=%d", conf.Name, conf.Age)
	}
}

func TestInjectByFlag_InitTagEnv(t *testing.T) {
	t.Setenv("NAME_TAG", "fromenv")
	gc := newGlobal[UserConfig, EmbeddedPresets]()
	conf := &injectInitTagged{}
	if err := gc.InjectByFlag([]string{"prog"}, conf); err != nil {
		t.Fatal(err)
	}
	if conf.Name != "fromenv" {
		t.Fatalf("Name=%q want fromenv (env beats default)", conf.Name)
	}
}

func TestInjectByFlag_InitTagSkip(t *testing.T) {
	t.Setenv("AUTO", "nope")
	gc := newGlobal[UserConfig, EmbeddedPresets]()
	conf := &injectInitTagged{}
	if err := gc.InjectByFlag([]string{"prog"}, conf); err != nil {
		t.Fatal(err)
	}
	if conf.Auto != "" {
		t.Fatalf("Auto=%q want empty: init:\"-\" skips env/flag binding", conf.Auto)
	}
}
