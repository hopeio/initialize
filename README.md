# initialize

[![Go Reference](https://pkg.go.dev/badge/github.com/hopeio/initialize.svg)](https://pkg.go.dev/github.com/hopeio/initialize)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Reflection-based bootstrap for **config** and **DAO** — declare fields, get wired clients.

基于反射的 **配置** 与 **DAO** 启动框架——声明字段，自动注入连接。

![initialize](_assets/initialize.webp)

```bash
go get github.com/hopeio/initialize@latest
```

---

## English

### What is initialize?

**initialize** loads application configuration (local files and/or remote centers), then reflectively constructs data-access objects (Redis, GORM, message queues, …) from struct fields.

You define `Config` and `Dao` types, call `NewGlobal`, and get a single process-wide handle with lifecycle hooks and cleanup.

It does **not** dictate how you serve HTTP or gRPC. It only owns bootstrap: config → inject → ready.

### Features

- **Root config** — app name, environment (`dev` / `test` / `prod` / custom), paths to local files and config centers (not hot-reloaded)
- **Local multi-file** — merge several paths; optional filesystem watch
- **Remote centers** — Nacos, Apollo, etcd, HTTP; register your own
- **Formats** — everything Viper supports (JSON, TOML, YAML, INI, dotenv, …)
- **Overrides** — environment variables and command-line flags via struct tags
- **DAO plugins (`contrib/`)** — GORM (MySQL / Postgres / SQLite), Redis, Kafka (sarama / confluent), NATS, NSQ, etcd, Elasticsearch, MinIO, MQTT, Badger, Pebble, Ristretto, InfluxDB, …
- **Templates** — generate a config skeleton from your structs
- **Hooks** — `BeforeInject` / `AfterInjectConfig` / `AfterInject` (and `*WithRoot` variants)
- **Cleanup** — `Cleanup()` and `Defer(fn)` for orderly shutdown

### Quick start

```bash
go run _example/main.go -c _example/config/config.toml
```

```go
package global

import (
	"time"

	"github.com/hopeio/initialize"
	"github.com/hopeio/initialize/contrib/gormdb/sqlite"
	initredis "github.com/hopeio/initialize/contrib/redis"
)

type config struct {
	initialize.EmbeddedPresets
	Customize struct {
		TokenMaxAge time.Duration
	}
}

func (c *config) BeforeInject() {
	c.Customize.TokenMaxAge = 24 * time.Hour
}

type dao struct {
	initialize.EmbeddedPresets
	GORMDB *sqlite.DB
	Redis  *initredis.Client
}

func (d *dao) AfterInject() {
	// tune pools, register callbacks, etc.
}

var Global = initialize.NewGlobal[*config, *dao]()
```

```go
func main() {
	defer Global.Cleanup()
	_ = Global.Config
	_ = Global.Dao
}
```

Minimal root TOML:

```toml
Name = "myapp"
Env = "dev"

[dev]
debug = true
ConfigTemplateDir = "."

[dev.localConfig]
Paths = ["local.toml"]
ReloadInterval = "1s"
```

Single-environment apps can skip `Env` and use `-c` / a local `config.*` file.

### Lifecycle

| Hook | When |
|------|------|
| `BeforeInject` (+ `WithRoot`) | Before unmarshalling into your structs |
| `AfterInjectConfig` (+ `WithRoot`) | Config ready; DAO `Init` not finished |
| `AfterInject` (+ `WithRoot`) | All DAO fields initialized |

Use `SkipInjectDaos` in root config to skip named fields when a dependency is unavailable locally.

### Writing a plugin

Implement `DaoField` (`Config()` / `Init()` / `Closer`) or the generic `DaoG` / `DaoConfig` helpers — same injection path as built-in contrib packages.

### License

[MIT](LICENSE)

---

## 中文

### initialize 是什么？

**initialize** 负责加载应用配置（本地文件和/或远程配置中心），再通过反射根据结构体字段构造数据访问对象（Redis、GORM、消息队列等）。

你定义 `Config` 与 `Dao`，调用 `NewGlobal`，即可得到带生命周期钩子与清理逻辑的全局句柄。

它**不**规定 HTTP / gRPC 怎么写，只负责启动链路：配置 → 注入 → 就绪。

### 特性

- **Root 配置** — 应用名、环境（`dev` / `test` / `prod` / 自定义）、本地路径与配置中心（根配置本身不热更）
- **本地多文件** — 多路径合并，可选文件监听
- **远程中心** — Nacos、Apollo、etcd、HTTP；可自行注册
- **格式** — Viper 支持的格式均可（JSON、TOML、YAML、INI、dotenv…）
- **覆盖** — 环境变量与命令行 flag（结构体 tag）
- **DAO 插件（`contrib/`）** — GORM（MySQL / Postgres / SQLite）、Redis、Kafka（sarama / confluent）、NATS、NSQ、etcd、Elasticsearch、MinIO、MQTT、Badger、Pebble、Ristretto、InfluxDB…
- **模板** — 根据结构体生成配置骨架
- **钩子** — `BeforeInject` / `AfterInjectConfig` / `AfterInject`（及 `*WithRoot` 变体）
- **清理** — `Cleanup()` 与 `Defer(fn)` 有序退出

### 快速开始

```bash
go run _example/main.go -c _example/config/config.toml
```

```go
package global

import (
	"time"

	"github.com/hopeio/initialize"
	"github.com/hopeio/initialize/contrib/gormdb/sqlite"
	initredis "github.com/hopeio/initialize/contrib/redis"
)

type config struct {
	initialize.EmbeddedPresets
	Customize struct {
		TokenMaxAge time.Duration
	}
}

func (c *config) BeforeInject() {
	c.Customize.TokenMaxAge = 24 * time.Hour
}

type dao struct {
	initialize.EmbeddedPresets
	GORMDB *sqlite.DB
	Redis  *initredis.Client
}

func (d *dao) AfterInject() {
	// 连接池、回调、业务侧初始化…
}

var Global = initialize.NewGlobal[*config, *dao]()
```

```go
func main() {
	defer Global.Cleanup()
	_ = Global.Config
	_ = Global.Dao
}
```

最小 Root TOML：

```toml
Name = "myapp"
Env = "dev"

[dev]
debug = true
ConfigTemplateDir = "."

[dev.localConfig]
Paths = ["local.toml"]
ReloadInterval = "1s"
```

单环境项目可省略 `Env`，使用 `-c` 或目录下的 `config.*`。

### 生命周期

| 钩子 | 时机 |
|------|------|
| `BeforeInject`（及 `WithRoot`） | 反序列化写入结构体之前 |
| `AfterInjectConfig`（及 `WithRoot`） | 配置已就绪，DAO `Init` 尚未完成 |
| `AfterInject`（及 `WithRoot`） | 全部 DAO 字段已初始化 |

Root 中的 `SkipInjectDaos` 可跳过本地暂不需要的字段。

### 自定义插件

实现 `DaoField`（`Config()` / `Init()` / `Closer`），或使用 `DaoG` / `DaoConfig` 辅助类型，即可走与内置 `contrib` 相同的注入路径。

### 许可证

[MIT](LICENSE)
