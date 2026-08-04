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
	Name string `flag:"name:name;short:n;default:test;usage:name;env:NAME"`
	Age  int    `flag:"name:age;short:a;default:18;usage:age;env:AGE"`
}

type UserSliceConfig struct {
	EmbeddedPresets
	Labels []string `flag:"name:labels;default:a,b;usage:labels"`
}

type UserDurationConfig struct {
	EmbeddedPresets
	Wait time.Duration `flag:"name:wait;default:1h;usage:wait"`
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

	gc := NewGlobalConfig[*UserConfig]()
	if gc.Config == nil {
		t.Fatal("nil Config")
	}
	if gc.Config.Name != "aaa" {
		t.Fatalf("Name not injected by flag, got=%q want=%q", gc.Config.Name, "aaa")
	}
	if gc.Config.Age != 12 {
		t.Fatalf("Age not injected by flag, got=%d want=%d", gc.Config.Age, 12)
	}
}

func TestNoConfigFileWithEnv(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test", "-e", "dev"}

	t.Setenv("NAME", "envname")
	t.Setenv("AGE", "21")

	gc := NewGlobalConfig[*UserConfig]()
	if gc.Config == nil {
		t.Fatal("nil Config")
	}
	if gc.Config.Name != "envname" {
		t.Fatalf("Name not injected by env, got=%q want=%q", gc.Config.Name, "envname")
	}
	if gc.Config.Age != 21 {
		t.Fatalf("Age not injected by env, got=%d want=%d", gc.Config.Age, 21)
	}
}

func TestNoConfigFile_Defaults(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	gc := NewGlobalConfig[*UserConfig]()
	if gc.Config == nil {
		t.Fatal("nil Config")
	}
	if gc.Config.Name != "test" {
		t.Fatalf("Name default not applied, got=%q want=%q", gc.Config.Name, "test")
	}
	if gc.Config.Age != 18 {
		t.Fatalf("Age default not applied, got=%d want=%d", gc.Config.Age, 18)
	}
}

func TestNoConfigFile_SliceDefaults(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	gc := NewGlobalConfig[*UserSliceConfig]()
	if gc.Config == nil {
		t.Fatal("nil Config")
	}
	if len(gc.Config.Labels) != 2 || gc.Config.Labels[0] != "a" || gc.Config.Labels[1] != "b" {
		t.Fatalf("Labels default not applied, got=%v want=%v", gc.Config.Labels, []string{"a", "b"})
	}
}

func TestNoConfigFile_DurationDefaults(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	gc := NewGlobalConfig[*UserDurationConfig]()
	if gc.Config == nil {
		t.Fatal("nil Config")
	}
	if gc.Config.Wait != time.Hour {
		t.Fatalf("Wait default not applied, got=%v want=%v", gc.Config.Wait, time.Hour)
	}
}
