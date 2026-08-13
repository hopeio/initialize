# initialize

[![Go Reference](https://pkg.go.dev/badge/github.com/hopeio/initialize.svg)](https://pkg.go.dev/github.com/hopeio/initialize)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[English](README.md)

```bash
go get github.com/hopeio/initialize@latest
```

**initialize** 用配置启动进程：加载设置、创建客户端（数据库、缓存、消息中间件…），并挂到带类型的全局句柄上。你用结构体字段描述*需要什么*，库负责*怎么建连*。

## 解决什么问题

`main` 里往往堆满 `viper.ReadInConfig`、`gorm.Open`、`redis.NewClient`、重试和清理。环境一多、配置进中心、热更新后补，代码会越来越散。

使用 initialize：

1. 写好根配置（应用名、环境、本地文件、远程中心）。
2. 声明 `Config` 与 `Dao` 结构体。
3. 调用 `NewGlobal[*Config, *Dao](...)`。
4. 使用 `Global.Conf()` / `Global.Dao`，并 `defer Global.Cleanup()`。

## 能力

- **根配置 vs 业务配置** — 根配置选环境与来源；业务字段来自本地和/或中心
- **多环境** — `dev` / `test` / `prod` 或自定义名
- **本地文件** — 多路径合并，可选重载间隔 / 监听
- **远程中心** — Nacos、Apollo、etcd、HTTP；可注册自定义实现
- **格式** — Viper 支持的格式（TOML、YAML、JSON、INI、dotenv…）
- **覆盖** — 结构体 tag 绑定 flag 与环境变量
- **DAO 插件** — 字段类型用 contrib 客户端，按嵌套配置 `Init`
- **生命周期** — `BeforeInject`、`AfterInjectConfig`、`AfterInject`（可选 `*WithRoot`）
- **模板** — 按结构体生成配置骨架
- **退出** — `Cleanup` 关资源；`Defer` 注册额外清理

## 跑示例

```bash
go run ./_example -c _example/config/config.toml
```

## 最小用法

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
	// GORM 回调、连接池等
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

### 根配置示意

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

业务键（`[HTTP]`、`[DB]`、`[Cache]`…）放在 `local.toml` 或远程文档。某环境不需要的字段可用 `SkipInjectDaos` 跳过。

单环境可省略 `Env`，用 `-c` 指定配置文件。

## 热更新与快照

开启 `Watch = true`（本地文件）或接入配置中心后，每次变更会构建一份**全新的 Config 快照**，走完整注入生命周期后原子替换，旧对象不会被原地修改：

- `Global.Conf()` — 无锁返回当前最新快照，默认入口；需要感知热更新的代码每次使用时调用。
- `Global.StartupConf()` — 启动快照，初始化后不再变化。仅在需要与进程实际启动值保持一致时使用（如监听地址）。

快照一经发布即不可变，请勿写入其字段。DAO（连接类资源）不参与热更新。

## 钩子

| 方法 | 时机 |
|------|------|
| `BeforeInject` / `BeforeInjectWithRoot` | 解码前设默认值 |
| `AfterInjectConfig` / `AfterInjectConfigWithRoot` | 配置已写入；DAO 尚未全部 Init |
| `AfterInject` / `AfterInjectWithRoot` | 全部 DAO 就绪 |

## Contrib 客户端

`contrib/` 下可直接当字段用的类型：

`gormdb`（mysql / postgres / sqlite）· `redis` · `sarama` · `confluent` · `nats` · `nsq` · `etcd` · `elasticsearch` · `minio` · `mqtt` · `badger` · `pebble` · `bbolt` · `ristretto` · `influxdb` · `duckdb` · `flightsql` · `mail` · `apollo` · `nacos` · `viper`

自定义：实现 `DaoField`（`Config` / `Init` / `Closer`），或使用 `DaoG` / `DaoConfig`。

## 配置中心

内置：本地多文件、Nacos、Apollo、etcd、HTTP。  
扩展：`RegisterConfigCenter`。

## 许可证

[MIT](LICENSE)
