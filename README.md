# initialize

[![Go Reference](https://pkg.go.dev/badge/github.com/hopeio/initialize.svg)](https://pkg.go.dev/github.com/hopeio/initialize)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[中文文档](README.zh-CN.md)

```bash
go get github.com/hopeio/initialize@latest
```

**initialize** boots your process from configuration: it loads settings, builds clients (databases, caches, brokers, …), and exposes them on a typed global handle. You describe *what* you need as struct fields; the library wires *how* they are created.

## Problem it solves

`main` often becomes a script of `viper.ReadInConfig`, `gorm.Open`, `redis.NewClient`, retry loops, and cleanup. Environments differ, secrets move to a config center, and hot reload is bolted on later.

With initialize you:

1. Define root settings (app name, env, local files, remote center).
2. Declare a `Config` struct and a `Dao` struct.
3. Call `NewGlobal[*Config, *Dao](...)`.
4. Use `Global.Config` / `Global.Dao` and `defer Global.Cleanup()`.

## Capabilities

- **Root vs business config** — root chooses environment and sources; business fields come from local files and/or a center
- **Environments** — `dev` / `test` / `prod` or any name you define
- **Local files** — multiple paths, optional reload interval / file watch
- **Remote centers** — Nacos, Apollo, etcd, HTTP; register custom implementations
- **Formats** — Viper codecs (TOML, YAML, JSON, INI, dotenv, …)
- **Overrides** — struct tags for flags and environment variables
- **DAO plugins** — drop fields typed as contrib clients; they `Init` from nested config
- **Lifecycle hooks** — `BeforeInject`, `AfterInjectConfig`, `AfterInject` (optional `*WithRoot`)
- **Template generation** — emit a config skeleton from your structs
- **Shutdown** — `Cleanup` closes resources; `Defer` registers extra teardown

## Try the example

```bash
go run ./_example -c _example/config/config.toml
```

## Minimal usage

```go
package global

import (
	"time"

	"github.com/hopeio/initialize"
	"github.com/hopeio/initialize/contrib/gormdb/postgres"
	initredis "github.com/hopeio/initialize/contrib/redis"
)

type Config struct {
	initialize.EmbeddedPresets
	HTTP struct {
		ReadTimeout time.Duration
	}
}

func (c *Config) BeforeInject() {
	if c.HTTP.ReadTimeout == 0 {
		c.HTTP.ReadTimeout = 5 * time.Second
	}
}

type Dao struct {
	initialize.EmbeddedPresets
	DB    *postgres.DB
	Cache *initredis.Client
}

func (d *Dao) AfterInject() {
	// register GORM callbacks, tune pools, …
}

var Global = initialize.NewGlobal[*Config, *Dao]()
```

```go
func main() {
	defer global.Global.Cleanup()
	db := global.Global.Dao.DB
	_ = db
}
```

### Root config sketch

```toml
Name = "orders"
Env = "dev"

[dev]
debug = true

[dev.localConfig]
Paths = ["local.toml"]
ReloadInterval = "2s"

[dev.ConfigCenter]
Format = "toml"
Type = "nacos"
```

Business keys (`[HTTP]`, `[DB]`, `[Cache]`, …) live in `local.toml` or the remote document. Use `SkipInjectDaos` when a field should be skipped in a given environment.

Single-env apps can omit `Env` and pass `-c path/to/config.toml`.

## Hot reload & snapshots

With `Watch = true` (local files) or a config center attached, every change builds a **brand-new Config snapshot**, runs the full injection lifecycle on it, and publishes it atomically — the old object is never mutated in place:

- `Global.Config` — the startup snapshot; it never changes after init and is safe to hold long-term.
- `Global.Conf()` — returns the latest snapshot lock-free; call it on each use when you need to observe reloads.

Published snapshots are immutable: never write to their fields. DAOs (connection-like resources) do not participate in hot reload.

## Hooks

| Method | Moment |
|--------|--------|
| `BeforeInject` / `BeforeInjectWithRoot` | Defaults before decode |
| `AfterInjectConfig` / `AfterInjectConfigWithRoot` | Config filled; DAO init pending |
| `AfterInject` / `AfterInjectWithRoot` | All DAO fields ready |

## Contrib clients

Ship-ready field types under `contrib/`:

`gormdb` (mysql / postgres / sqlite) · `redis` · `sarama` · `confluent` · `nats` · `nsq` · `etcd` · `elasticsearch` · `minio` · `mqtt` · `badger` · `pebble` · `bbolt` · `ristretto` · `influxdb` · `duckdb` · `flightsql` · `mail` · `apollo` · `nacos` · `viper`

Custom type: implement `DaoField` (`Config` / `Init` / `Closer`) or use `DaoG` / `DaoConfig`.

## Config centers

Built-in: local multi-file, Nacos, Apollo, etcd, HTTP.  
Extend with `RegisterConfigCenter`.

## License

[MIT](LICENSE)
