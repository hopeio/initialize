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

	g := NewGlobalConfig[UserConfig]()
	if g.Conf() == nil {
		t.Fatal("nil Config")
	}
	if g.Conf().Name != "aaa" {
		t.Fatalf("Name not injected by flag, got=%q want=%q", g.Conf().Name, "aaa")
	}
	if g.Conf().Age != 12 {
		t.Fatalf("Age not injected by flag, got=%d want=%d", g.Conf().Age, 12)
	}
}

func TestNoConfigFileWithEnv(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test", "-e", "dev"}

	t.Setenv("NAME", "envname")
	t.Setenv("AGE", "21")

	g := NewGlobalConfig[UserConfig]()
	if g.Conf() == nil {
		t.Fatal("nil Config")
	}
	if g.Conf().Name != "envname" {
		t.Fatalf("Name not injected by env, got=%q want=%q", g.Conf().Name, "envname")
	}
	if g.Conf().Age != 21 {
		t.Fatalf("Age not injected by env, got=%d want=%d", g.Conf().Age, 21)
	}
}

func TestNoConfigFile_Defaults(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	g := NewGlobalConfig[UserConfig]()
	if g.Conf() == nil {
		t.Fatal("nil Config")
	}
	if g.Conf().Name != "test" {
		t.Fatalf("Name default not applied, got=%q want=%q", g.Conf().Name, "test")
	}
	if g.Conf().Age != 18 {
		t.Fatalf("Age default not applied, got=%d want=%d", g.Conf().Age, 18)
	}
}

func TestNoConfigFile_SliceDefaults(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	g := NewGlobalConfig[UserSliceConfig]()
	if g.Conf() == nil {
		t.Fatal("nil Config")
	}
	if len(g.Conf().Labels) != 2 || g.Conf().Labels[0] != "a" || g.Conf().Labels[1] != "b" {
		t.Fatalf("Labels default not applied, got=%v want=%v", g.Conf().Labels, []string{"a", "b"})
	}
}

func TestNoConfigFile_DurationDefaults(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	g := NewGlobalConfig[UserDurationConfig]()
	if g.Conf() == nil {
		t.Fatal("nil Config")
	}
	if g.Conf().Wait != time.Hour {
		t.Fatalf("Wait default not applied, got=%v want=%v", g.Conf().Wait, time.Hour)
	}
}
