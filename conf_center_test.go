/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type stubConfigCenter struct {
	typ string
}

func (s *stubConfigCenter) Type() string { return s.typ }

func (s *stubConfigCenter) Config() any { return s }

func (s *stubConfigCenter) Close() error { return nil }

func (s *stubConfigCenter) Handle(_ context.Context, _ func(io.Reader) error, _ func(io.Reader) error) error {
	return nil
}

func TestRegisterConfigCenter_Nil(t *testing.T) {
	a := newApp[UserConfig, EmbeddedPresets]()
	a.registerConfigCenters([]ConfigCenter{nil})
	if len(a.registeredCenters()) != 0 {
		t.Fatal("nil center should not be registered")
	}
}

func TestRegisterConfigCenter_GetAndCopy(t *testing.T) {
	a := newApp[UserConfig, EmbeddedPresets]()
	cc := &stubConfigCenter{typ: "stubcc"}
	a.registerConfigCenters([]ConfigCenter{cc})

	got := a.getCenter("stubcc")
	if got != ConfigCenter(cc) {
		t.Fatal("getCenter want registered instance")
	}

	cp := a.registeredCenters()
	if cp["stubcc"] != ConfigCenter(cc) {
		t.Fatal("copy map should contain registered center")
	}
	cp["stubcc"] = nil
	if a.getCenter("stubcc") != ConfigCenter(cc) {
		t.Fatal("registeredCenters should return a copy")
	}

	a.registerConfigCenters([]ConfigCenter{&stubConfigCenter{typ: "stubcc"}})
	if a.getCenter("stubcc") != ConfigCenter(cc) {
		t.Fatal("duplicate registration should not replace existing")
	}
}

func TestLocal_TypeAndConfig(t *testing.T) {
	ld := &Local{Watch: true, Paths: []string{"/tmp/a.toml"}}
	if ld.Type() != "local" {
		t.Fatalf("Type=%q want local", ld.Type())
	}
	if ld.Config() != ld {
		t.Fatal("Config() should return self")
	}
}

func TestLocal_Handle_EmptyPaths(t *testing.T) {
	ld := &Local{}
	err := ld.Handle(context.Background(), nil, nil)
	if err == nil || err.Error() != "empty local config path" {
		t.Fatalf("Handle empty paths: err=%v", err)
	}
}

func TestLocal_Handle_LoadsFileWithoutWatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "local.toml")
	const want = `Name = "fromlocal"`
	if err := os.WriteFile(path, []byte(want), 0644); err != nil {
		t.Fatal(err)
	}

	ld := &Local{Paths: []string{path}, Watch: false}
	var merged string
	err := ld.Handle(context.Background(), func(r io.Reader) error {
		b, readErr := io.ReadAll(r)
		if readErr != nil {
			return readErr
		}
		merged = string(b)
		return nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(merged, "fromlocal") {
		t.Fatalf("merge content=%q want substring fromlocal", merged)
	}
	if ld.watcher != nil {
		t.Fatal("Watch=false should not create watcher")
	}
}

func TestLocal_Handle_MissingFile(t *testing.T) {
	ld := &Local{Paths: []string{filepath.Join(t.TempDir(), "missing.toml")}}
	err := ld.Handle(context.Background(), func(io.Reader) error { return nil }, nil)
	if err == nil {
		t.Fatal("Handle want error for missing file")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err=%v want os.ErrNotExist", err)
	}
}

func TestLocal_Close_NoWatcher(t *testing.T) {
	ld := &Local{}
	if err := ld.Close(); err != nil {
		t.Fatalf("Close without watcher: %v", err)
	}
}
