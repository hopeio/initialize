/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"testing"
	"time"

	"github.com/go-viper/mapstructure/v2"
)

type decodeTarget struct {
	Port   int
	Wait   time.Duration
	Labels []string
}

type decodeSquashInner struct {
	Port int `mapstructure:"port"`
}

type decodeSquashOuter struct {
	decodeSquashInner `mapstructure:",squash"`
	Name              string `mapstructure:"name"`
}

func TestDecode_WeakTypedPort(t *testing.T) {
	var dst decodeTarget
	err := Decode(&dst, map[string]any{"port": "8080"})
	if err != nil {
		t.Fatal(err)
	}
	if dst.Port != 8080 {
		t.Fatalf("Port=%d want 8080", dst.Port)
	}
}

func TestDecode_DurationString(t *testing.T) {
	var dst decodeTarget
	err := Decode(&dst, map[string]any{"wait": "1h"})
	if err != nil {
		t.Fatal(err)
	}
	if dst.Wait != time.Hour {
		t.Fatalf("Wait=%v want 1h", dst.Wait)
	}
}

func TestDecode_SliceCommaString(t *testing.T) {
	var dst decodeTarget
	err := Decode(&dst, map[string]any{"labels": "a,b"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b"}
	if len(dst.Labels) != len(want) {
		t.Fatalf("Labels=%v want %v", dst.Labels, want)
	}
	for i := range want {
		if dst.Labels[i] != want[i] {
			t.Fatalf("Labels[%d]=%q want %q", i, dst.Labels[i], want[i])
		}
	}
}

func TestDecode_SquashEmbedded(t *testing.T) {
	var dst decodeSquashOuter
	err := Decode(&dst, map[string]any{
		"port": "9090",
		"name": "svc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if dst.Port != 9090 {
		t.Fatalf("Port=%d want 9090", dst.Port)
	}
	if dst.Name != "svc" {
		t.Fatalf("Name=%q want svc", dst.Name)
	}
}

func TestDecode_AllHooksCombined(t *testing.T) {
	var dst decodeTarget
	err := Decode(&dst, map[string]any{
		"port":   "3000",
		"wait":   "30m",
		"labels": "x,y,z",
	})
	if err != nil {
		t.Fatal(err)
	}
	if dst.Port != 3000 {
		t.Fatalf("Port=%d want 3000", dst.Port)
	}
	if dst.Wait != 30*time.Minute {
		t.Fatalf("Wait=%v want 30m", dst.Wait)
	}
	if len(dst.Labels) != 3 || dst.Labels[0] != "x" || dst.Labels[2] != "z" {
		t.Fatalf("Labels=%v want [x y z]", dst.Labels)
	}
}

func TestDefaultDecoderConfig_WithOption(t *testing.T) {
	var dst decodeTarget
	opt := func(c *mapstructure.DecoderConfig) {
		c.WeaklyTypedInput = false
	}
	cfg := defaultDecoderConfig(&dst, opt)
	if cfg.WeaklyTypedInput {
		t.Fatal("WeaklyTypedInput want false after option")
	}
	if !cfg.Squash {
		t.Fatal("Squash want true by default")
	}
}
