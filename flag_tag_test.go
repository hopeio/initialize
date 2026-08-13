/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"testing"
	"time"
)

type injectFlagTagged struct {
	Name string        `flag:"name:name;short:n;default:def;usage:name"`
	Age  int           `flag:"name:age;default:18;usage:age"`
	Wait time.Duration `flag:"name:wait;default:1s;usage:wait"`
}

type injectFlagDefaultEnv struct {
	Port  int
	Debug bool
}

type injectFlagNested struct {
	Server injectFlagDefaultEnv
}

func TestInjectByFlag_OverridesDefaults(t *testing.T) {
	gc := newGlobal[UserConfig, EmbeddedPresets]()
	conf := &injectFlagTagged{}
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
	conf := &injectFlagTagged{}
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

func TestInjectByFlag_DefaultEnvWithoutFlagTag(t *testing.T) {
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
