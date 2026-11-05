/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hopeio/gox/encoding"
)

type struct2MapSample struct {
	Name   string
	Port   int
	Labels []string
	Nested struct {
		Enabled bool
	}
}

type Struct2MapSquashInner struct {
	Count int
}

type struct2MapSquashOuter struct {
	Struct2MapSquashInner
	Title string
}

// mustStruct2Map keeps these cases readable: struct2Map now reports a malformed init
// tag instead of exiting, and a test that ignored that error would pass on a
// half-written map.
func mustStruct2Map(t *testing.T, v any, m map[string]any) {
	t.Helper()
	if err := struct2Map(v, m); err != nil {
		t.Fatalf("struct2Map: %v", err)
	}
}

func TestStruct2Map_BasicFields(t *testing.T) {
	src := &struct2MapSample{
		Name:   "demo",
		Port:   8080,
		Labels: []string{"a", "b"},
	}
	src.Nested.Enabled = true

	m := make(map[string]any)
	mustStruct2Map(t, src, m)

	if m["Name"] != "demo" {
		t.Fatalf("Name=%v want demo", m["Name"])
	}
	if m["Port"] != 8080 {
		t.Fatalf("Port=%v want 8080", m["Port"])
	}
	labels, ok := m["Labels"].([]any)
	if !ok {
		t.Fatalf("Labels=%T want []any", m["Labels"])
	}
	if len(labels) != 2 || labels[0] != "a" || labels[1] != "b" {
		t.Fatalf("Labels=%v want [a b]", labels)
	}
	nested, ok := m["Nested"].(map[string]any)
	if !ok || nested["Enabled"] != true {
		t.Fatalf("Nested=%v want Enabled=true", m["Nested"])
	}
}

func TestStruct2Map_EmptySliceCreatesTemplateElement(t *testing.T) {
	type item struct {
		Name string
	}
	type sliceHolder struct {
		Items []*item
	}
	m := make(map[string]any)
	mustStruct2Map(t, &sliceHolder{}, m)
	items, ok := m["Items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("Items=%v want one template element", m["Items"])
	}
	if _, ok := items[0].(map[string]any); !ok {
		t.Fatalf("Items[0]=%T want map template", items[0])
	}
}

func TestStruct2Map_NonNilPointerField(t *testing.T) {
	type inner struct {
		Name string
	}
	type outer struct {
		Inner *inner
	}
	src := &outer{Inner: &inner{Name: "x"}}
	m := make(map[string]any)
	// Non-nil pointer fields used to recurse forever and overflow the stack.
	mustStruct2Map(t, src, m)
	innerMap, ok := m["Inner"].(map[string]any)
	if !ok || innerMap["Name"] != "x" {
		t.Fatalf("Inner=%v want map with Name=x", m["Inner"])
	}
}

func TestStruct2Map_SquashAnonymous(t *testing.T) {
	src := &struct2MapSquashOuter{
		Struct2MapSquashInner: Struct2MapSquashInner{Count: 3},
		Title:                 "t",
	}
	m := make(map[string]any)
	mustStruct2Map(t, src, m)
	if m["Count"] != 3 {
		t.Fatalf("Count=%v want 3 (squash)", m["Count"])
	}
	if m["Title"] != "t" {
		t.Fatalf("Title=%v want t", m["Title"])
	}
}

func TestRegisterUnSupportTemplateTypes(t *testing.T) {
	before := append([]string(nil), unSupportTemplateTypes...)
	t.Cleanup(func() { unSupportTemplateTypes = before })

	RegisterUnSupportTemplateTypes("custom.Type")
	if !containsString(unSupportTemplateTypes, "custom.Type") {
		t.Fatal("RegisterUnSupportTemplateTypes should append type")
	}
}

func TestFormatDecoderConfigOption_YmlAlias(t *testing.T) {
	opts := formatDecoderConfigOption(encoding.Yml)
	if len(opts) == 0 {
		t.Fatal("formatDecoderConfigOption returned no options")
	}
}

func TestDaoConfig2Map_EmbeddedDaoG(t *testing.T) {
	type daoHolder struct {
		EmbeddedPresets
		DB DaoG[*daoOkConfig, daoTestClient]
	}
	holder := &daoHolder{
		DB: DaoG[*daoOkConfig, daoTestClient]{Conf: &daoOkConfig{}},
	}
	m := make(map[string]any)
	daoConfig2Map(reflect.ValueOf(holder).Elem(), m)
	dbMap, ok := m["DB"].(map[string]any)
	if !ok {
		t.Fatalf("DB=%v want map", m["DB"])
	}
	_ = dbMap
}

func TestGenConfigTemplate_WritesLocalTemplate(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(configPath, []byte(`Name = "cfg"`), 0644); err != nil {
		t.Fatal(err)
	}
	tmplDir := filepath.Join(dir, "tmpl")
	os.Args = []string{"test", "-c", configPath, "--conf_tmpl_dir", tmplDir}

	a := NewConfig[UserConfig]()
	mustStartT(t, a)
	wantFile := filepath.Join(tmplDir, prefixLocalTemplate+"toml")
	if _, err := os.Stat(wantFile); err != nil {
		t.Fatalf("template file missing: %v", err)
	}
	if a.Conf() == nil {
		t.Fatal("nil Config")
	}
}

func TestGenConfigTemplate_SingleFileMode(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(configPath, []byte(`Name = "single"`), 0644); err != nil {
		t.Fatal(err)
	}
	tmplDir := filepath.Join(dir, "tmpl")
	os.Args = []string{"test", "-c", configPath, "--conf_tmpl_dir", tmplDir, "-e", ""}

	a := NewConfig[UserDurationConfig]()
	mustStartT(t, a)
	wantFile := filepath.Join(tmplDir, prefixConfigTemplate+"toml")
	if _, err := os.Stat(wantFile); err != nil {
		t.Fatalf("single template file missing: %v", err)
	}
	if a.Conf().Wait != time.Hour {
		t.Fatalf("Wait=%v want default 1h", a.Conf().Wait)
	}
}

func containsString(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}
