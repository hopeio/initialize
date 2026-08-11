# initialize

[![Go Reference](https://pkg.go.dev/badge/github.com/hopeio/initialize.svg)](https://pkg.go.dev/github.com/hopeio/initialize)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Config & DAO, injected. Not configured by hand.**  
基于反射的配置与数据访问对象（DAO）初始化框架：本地多文件 + 远程配置中心 + env/flag，一行 `NewGlobal` 拉起整站依赖。

![initialize](_assets/initialize.webp)

```bash
go get github.com/hopeio/initialize@latest
```

## 解决什么问题

手写 `viper.ReadInConfig` → 解析 → `redis.NewClient` → `gorm.Open` → 再拼 OTel，容易散落在 `main` 里、难以分环境、更难热更新。

**initialize** 把这件事收成固定生命周期：

1. 读 **RootConfig**（应用名、环境、本地路径、配置中心）
2. 合并本地 / 远程配置（Viper 全格式）
3. 反射注入 **Config** 与 **Dao** 字段
4. 调用 `BeforeInject` / `AfterInject*` 钩子
5. `Cleanup` / `Defer` 优雅释放

你只声明结构体字段；连接怎么建，交给 `contrib/*` 插件。

## 特性

- **本地多配置** — 多路径合并，可选 `fsnotify` 热重载
- **环境隔离** — `dev` / `test` / `prod`（可自定义），按 `Env` 选段
- **env + flag** — 结构体 tag 覆盖配置项
- **远程配置中心** — Nacos / Apollo / etcd / HTTP，可 `RegisterConfigCenter` 扩展
- **格式自由** — json、toml、yaml、ini、dotenv、hcl…（Viper 支持的均可）
- **DAO 插件生态** — GORM（MySQL/Postgres/SQLite）、Redis、Kafka、NATS、NSQ、etcd、ES、MinIO、MQTT、Badger、Pebble、Ristretto…
- **模板生成** — 最小 Root 配置启动即可吐出业务配置骨架
- **边界清晰** — **不**绑定 HTTP/gRPC 框架，**不**侵入 DAO 塞业务观测字段；OTel 等请在 `AfterInject` 用客户端原生钩子挂载

## 30 秒上手

```bash
go run _example/main.go -c _example/config/config.toml
```

### 声明 Config / Dao

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
	// 在此挂 GORM 插件、OTel、连接池调优等
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

### Root 配置（不热更）

```toml
Name = "myapp"
Env = "dev"

[dev]
debug = true
ConfigTemplateDir = "."

[dev.localConfig]
Paths = ["local.toml"]
ReloadInterval = "1s"

[dev.ConfigCenter]
Format = "toml"
Type = "nacos"
```

单环境项目可省略 `Env`，放 `config.toml` 或 `-c` 指定路径。

## 生命周期钩子

| 接口 | 时机 |
|------|------|
| `BeforeInject` / `BeforeInjectWithRoot` | 注入前（可设默认值） |
| `AfterInjectConfig` / `*WithRoot` | 配置已解析、DAO `Init` 前 |
| `AfterInject` / `AfterInjectWithRoot` | DAO 全部就绪（挂插件、补业务初始化） |

`SkipInjectDaos` 可跳过指定 DAO 字段（例如本地不启 Apollo）。

## contrib 一览

`apollo` · `badger` · `bbolt` · `confluent` · `duckdb` · `elasticsearch` · `etcd` · `flightsql` · `gormdb` · `influxdb` · `mail` · `minio` · `mqtt` · `nacos` · `nats` · `nsq` · `pebble` · `redis` · `ristretto` · `sarama` · `viper`

自定义：实现 `DaoField`（`Config()` / `Init()` / `Closer`）即可接入同一套注入。

## 与生态如何协作

```
initialize  ──注入──►  Config / Dao（Redis、GORM…）
     │
     │ AfterInject 里挂 OTel / 业务插件
     ▼
   mix.Server.Run()     ← HTTP + gRPC 运行时
     │
   gox                  ← 日志、HTTP 工具、ID…
protobuf / protogen     ← 接口定义与代码生成
```

| 仓库 | 关系 |
|------|------|
| [gox](https://github.com/hopeio/gox) | 日志等基础能力；initialize 可依赖 |
| [mix](https://github.com/hopeio/mix) | 服务运行时；`Server` 可实现 Inject 钩子，由 initialize 拉起 |
| [protobuf](https://github.com/hopeio/protobuf) | 与 initialize **无直接依赖**；各自服务契约与启动配置 |

## 关键 API

| API | 说明 |
|-----|------|
| `NewGlobal` / `NewGlobalWith` / `NewGlobalConfig` | 创建全局容器 |
| `Cleanup` / `Defer` | 关闭资源 / 注册退出回调 |
| `RootConfig` | 启动根配置 |
| `RegisterConfigCenter` | 扩展远程配置中心 |
| `DaoField` / `DaoG` | DAO 插件契约 |

详见 [pkg.go.dev/github.com/hopeio/initialize](https://pkg.go.dev/github.com/hopeio/initialize) 与 [`_example/`](_example/)。

## License

[MIT](LICENSE) · Copyright © hopeio
