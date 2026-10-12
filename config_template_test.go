/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"github.com/hopeio/gox/encoding"
	"os"
	"testing"
	"time"
)

type UserConfig struct {
	EmbeddedPresets
	Name string `init:"flag:name;short_flag:n;default:test;usage:name;env:NAME"`
	Age  int    `init:"flag:age;short_flag:a;default:18;usage:age;env:AGE"`
}

type UserSliceConfig struct {
	EmbeddedPresets
	Labels []string `init:"flag:labels;default:a,b;usage:labels"`
}

type UserDurationConfig struct {
	EmbeddedPresets
	Wait time.Duration `init:"flag:wait;default:1h;usage:wait"`
}

func TestGenConfigTemplate(t *testing.T) {
	type args struct {
		format encoding.Format
		config Config
		dao    Dao
	}
}

func TestNoConfigFile(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test", "-n", "aaa", "-a", "12"}

	a := NewConfig[UserConfig]()
	a.Start()
	if a.Conf() == nil {
		t.Fatal("nil Config")
	}
	if a.Conf().Name != "aaa" {
		t.Fatalf("Name not injected by flag, got=%q want=%q", a.Conf().Name, "aaa")
	}
	if a.Conf().Age != 12 {
		t.Fatalf("Age not injected by flag, got=%d want=%d", a.Conf().Age, 12)
	}
}

func TestNoConfigFileWithEnv(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test", "-e", "dev"}

	t.Setenv("NAME", "envname")
	t.Setenv("AGE", "21")

	a := NewConfig[UserConfig]()
	a.Start()
	if a.Conf() == nil {
		t.Fatal("nil Config")
	}
	if a.Conf().Name != "envname" {
		t.Fatalf("Name not injected by env, got=%q want=%q", a.Conf().Name, "envname")
	}
	if a.Conf().Age != 21 {
		t.Fatalf("Age not injected by env, got=%d want=%d", a.Conf().Age, 21)
	}
}

func TestNoConfigFile_Defaults(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	a := NewConfig[UserConfig]()
	a.Start()
	if a.Conf() == nil {
		t.Fatal("nil Config")
	}
	if a.Conf().Name != "test" {
		t.Fatalf("Name default not applied, got=%q want=%q", a.Conf().Name, "test")
	}
	if a.Conf().Age != 18 {
		t.Fatalf("Age default not applied, got=%d want=%d", a.Conf().Age, 18)
	}
}

func TestNoConfigFile_SliceDefaults(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	a := NewConfig[UserSliceConfig]()
	a.Start()
	if a.Conf() == nil {
		t.Fatal("nil Config")
	}
	if len(a.Conf().Labels) != 2 || a.Conf().Labels[0] != "a" || a.Conf().Labels[1] != "b" {
		t.Fatalf("Labels default not applied, got=%v want=%v", a.Conf().Labels, []string{"a", "b"})
	}
}

func TestNoConfigFile_DurationDefaults(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	a := NewConfig[UserDurationConfig]()
	a.Start()
	if a.Conf() == nil {
		t.Fatal("nil Config")
	}
	if a.Conf().Wait != time.Hour {
		t.Fatalf("Wait default not applied, got=%v want=%v", a.Conf().Wait, time.Hour)
	}
}
