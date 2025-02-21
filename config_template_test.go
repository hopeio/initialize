/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"github.com/hopeio/utils/encoding"
	"os"
	"testing"
)

type UserConfig struct {
	EmbeddedPresets
	Name string `flag:"name:name;short:n;default:test;usage:name;env:NAME"`
	Age  int    `flag:"name:age;short:a;default:18;usage:age;env:AGE"`
}

func TestGenConfigTemplate(t *testing.T) {
	type args struct {
		format encoding.Format
		config Config
		dao    Dao
	}
}

func TestNoConfigFile(t *testing.T) {
	os.Args = []string{"test", "-n", "aaa", "-a", "12"}
	gc := NewGlobal2[*UserConfig]()
	t.Log(gc.Config)
}

func TestNoConfigFileWithEnv(t *testing.T) {
	os.Args = []string{"test", "-e", "dev", "-n", "aaa", "-a", "12"}
	gc := NewGlobal2[*UserConfig]()
	t.Log(gc.Config)
}
