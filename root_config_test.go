/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"path/filepath"
	"testing"
)

func TestEnvConfig_ResolveLocalPaths_Abs(t *testing.T) {
	rel := filepath.Join(".", "local.toml")
	ec := &EnvConfig{
		LocalConfig: Local{Paths: []string{rel}},
	}
	if err := ec.resolveLocalPaths(); err != nil {
		t.Fatal(err)
	}
	if len(ec.LocalConfig.Paths) != 1 {
		t.Fatalf("Paths len=%d want 1", len(ec.LocalConfig.Paths))
	}
	got := ec.LocalConfig.Paths[0]
	if !filepath.IsAbs(got) {
		t.Fatalf("Paths[0]=%q want absolute path", got)
	}
	if filepath.Base(got) != "local.toml" {
		t.Fatalf("Paths[0] base=%q want local.toml", filepath.Base(got))
	}
	// boot and setEnvConfig both call it; a second pass must be a no-op.
	if err := ec.resolveLocalPaths(); err != nil {
		t.Fatal(err)
	}
	if ec.LocalConfig.Paths[0] != got {
		t.Fatalf("second pass changed the path: %q -> %q", got, ec.LocalConfig.Paths[0])
	}
}
