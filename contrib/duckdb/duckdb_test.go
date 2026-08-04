/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package duckdb

import (
	"path/filepath"
	"testing"
)

func TestDuckDB(t *testing.T) {
	config := Config{
		DSN:         filepath.Join(t.TempDir(), "duck.db") + "?access_mode=read_write&threads=4",
	}
	db, err := config.Build()
	if err != nil {
		t.Fatal("Build err", err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE people (id INTEGER, name VARCHAR)`)
	if err != nil {
		t.Fatal("Exec err", err)
	}
}
