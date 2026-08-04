/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"net/http"
	"path/filepath"
	"testing"
)

func TestEnvConfig_AfterInject_AbsPaths(t *testing.T) {
	rel := filepath.Join(".", "local.toml")
	ec := &EnvConfig{
		LocalConfig: Local{Paths: []string{rel}},
	}
	ec.AfterInject()
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
}

func TestEnvConfig_AfterInject_InvalidProxyIgnored(t *testing.T) {
	oldTransport := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = oldTransport })

	ec := &EnvConfig{Proxy: "://bad"}
	ec.AfterInject()
	if http.DefaultClient.Transport != oldTransport {
		t.Fatal("invalid proxy should not change DefaultClient transport")
	}
}

func TestEnvConfig_AfterInject_ValidProxy(t *testing.T) {
	oldTransport := http.DefaultClient.Transport
	t.Cleanup(func() {
		http.DefaultClient.Transport = oldTransport
	})

	ec := &EnvConfig{Proxy: "http://127.0.0.1:8080"}
	ec.AfterInject()
	tr, ok := http.DefaultClient.Transport.(*http.Transport)
	if !ok || tr.Proxy == nil {
		t.Fatal("valid proxy should set DefaultClient transport with Proxy")
	}
}
