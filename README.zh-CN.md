# Clean SaaS Starter

[English](README.md) | 简体中文

一个开源的多租户 SaaS 脚手架，用于在可复用的内核之上构建新系统，而不是从某个具体业务项目开始演化。

## 1. 这是什么

本仓库是一个通用 SaaS 脚手架，不是某个垂直业务应用。

它的目标是为新建 SaaS 系统提供开箱即用的内核能力：

- 多租户上下文与隔离
- 认证与 RBAC
- 运行时配置、日志与中间件
- 审计与文件上传基础能力
- 项目脚手架与模块生成 CLI

当前核心技术栈：

- Go
- Echo
- sqlc
- Casbin
- PostgreSQL
- MinIO（可选，只有启用文件存储能力时才需要）

认证策略：

- 登录二次验证在框架层是可选能力
- 默认情况下，`/api/v1/auth/login` 在账号密码校验通过后可以直接返回 token
- 当 `auth.login_second_factor_enabled=true` 时，登录流程切换为 `login -> verify` 的 OTP challenge 模式
- 二次验证的投递方式可由使用者自行实现为短信、邮件、TOTP 等 adapter

数据库策略：

- PostgreSQL 是更推荐的生产路径
- PostgreSQL 可以通过 RLS 提供额外一层数据库级租户隔离保护
- SQLite 面向本地开发、演示和低门槛上手
- SQLite 不提供 PostgreSQL 这种数据库级 RLS 保护
- 在 SQLite 模式下，租户隔离必须依赖应用层和 repo 层的显式控制
- SQLite 启动链路会逐步补齐，当前主干运行路径仍然是 PostgreSQL

对象存储策略：

- 框架内置 MinIO 适配器
- 对象存储是可选能力，没有 OSS 也可以启动服务
- 主干不内置云厂商 SDK
- 阿里云、腾讯云、Cloudflare R2、AWS S3 等由使用者自行扩展 adapter
- 下载预签名应优先使用 `file_id` 作为稳定输入；`file_url` 目前仅作为兼容输入保留

当前 `oss` 配置项：

- `endpoint`
- `access_key`
- `secret_key`
- `bucket`
- `public_base_url`
- `use_ssl`

## 2. 如何使用

### 创建一个新项目

从当前脚手架生成一个新项目：

```bash
go run ./cmd/cli new-project --name my-saas --output ../my-saas
```

可选：

```bash
go run ./cmd/cli new-project --name my-saas --output ../my-saas --module-path github.com/acme/my-saas
```

生成的新项目会：

- 复制当前脚手架到目标目录
- 跳过 `.git`、`.env`、`app.yaml`、`build`、`logs` 等本地文件
- 替换默认模块名、二进制名、命令入口路径、示例数据库名等占位内容

### 启动生成后的项目

进入生成出来的项目目录后：

```bash
cp .env.example .env
make build
make dev
```

在执行 `make dev` 之前，请确认：

1. 数据库已经创建
2. 如果是 PostgreSQL，已经执行 `migrations/pgsql/` 里的 SQL
3. SQLite baseline 文件位于 `migrations/sqlite/`
4. `.env` 或 `app.yaml` 已正确指向数据库
5. 只有当你要启用文件上传/下载能力时，才需要继续配置 OSS
6. 只有当你的项目确实需要时，才开启登录二次验证

认证配置示例：

```yaml
auth:
  login_second_factor_enabled: false
```

或者在 `.env` 中：

```bash
AUTH_LOGIN_SECOND_FACTOR_ENABLED=false
```

### 在当前项目里生成一个模块

```bash
go run ./cmd/cli new-module --name post
```

可选：

```bash
go run ./cmd/cli new-module --name post --with-test
```

默认会生成：

```text
internal/domain/model/post.go
internal/domain/port/post.go
internal/app/usecase/post.go
internal/repo/pg/post_repo_pg.go
internal/delivery/http/handler/post_handler.go
db/query/post.sql
migrations/pgsql/<timestamp>_add_posts.sql
```

如果加上 `--with-test`，还会额外生成：

```text
internal/app/usecase/post_test.go
```

说明：

- `--name` 必须是 `snake_case`
- 输出路径是框架约定，不支持任意自定义
- 已存在的文件不会被覆盖

生成之后你还需要手工完成：

1. 在 migration 文件里补 DDL
2. 在 `db/query/<name>.sql` 里补 sqlc 查询
3. 完成生成出来的 `repo / usecase / handler`
4. 在模块准备好后，手工接入 bootstrap 和路由

## 3. 项目如何组织

本项目采用 clean architecture 风格，以内核优先的方式组织目录。

主要目录：

- `cmd/`
  - 服务入口与脚手架 CLI
- `internal/domain/`
  - 领域模型、错误、端口接口、认证上下文
- `internal/app/usecase/`
  - 应用用例
- `internal/repo/`
  - PostgreSQL、Casbin、MinIO、缓存、短信等基础设施适配器
- `internal/delivery/http/`
  - Echo handler、中间件、响应结构
- `internal/bootstrap/`
  - 组合根、配置加载、依赖注入、路由、应用启动
- `db/query/`
  - sqlc 查询定义
- `migrations/`
  - 数据库迁移，包含 PostgreSQL 和 SQLite baseline
- `docs/`
  - 设计文档和脚手架规划

关键文档：

- [docs/kernel-capability-boundary.md](docs/kernel-capability-boundary.md)
  - 内核与业务模块边界
- [docs/oss-optional-plan.md](docs/oss-optional-plan.md)
  - 先把对象存储改成可选能力，再进入 SQLite 支持
- [docs/sqlite-support-plan.md](docs/sqlite-support-plan.md)
  - 增加 SQLite 作为低门槛本地数据库路径
- [docs/file-capability-plan.md](docs/file-capability-plan.md)
  - 文件能力选配边界与演进方案
- [docs/scaffolding-cli-plan.md](docs/scaffolding-cli-plan.md)
  - `new-project` 与 `new-module` CLI 规划

这一部分主要面向愿意理解和改进这个脚手架本身的开发者。
