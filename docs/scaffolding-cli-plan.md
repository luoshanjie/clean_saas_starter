# Scaffolding CLI Plan

本文档用于定义本仓库脚手架 CLI 的两类能力边界：

- `new-project`
- `new-module`

目标不是一次把两者都做完，而是先把职责、输入输出、阶段性实现范围定清楚，避免后续 CLI 继续混成“项目联调工具”。

## 目标

脚手架 CLI 应服务两种不同层级的生成需求：

1. 生成一个完整的新 SaaS 项目
2. 在当前 SaaS 项目内生成一个业务模块骨架

这两类命令必须分开设计，因为它们的操作对象不同：

- `new-project` 操作的是“目录”
- `new-module` 操作的是“当前仓库内部结构”

## 命令一：new-project

### 职责

在指定目录生成一个新的 SaaS 项目骨架。

例如：

```bash
go run ./cmd/cli new-project --name my-saas --output ../my-saas
```

### 适用场景

- 新团队基于本仓库创建一个全新项目
- 想从内核模板得到一份独立可运行代码
- 类似 `create-tauri-app`、`create-next-app` 这类命令

### 第一阶段最小能力

第一阶段即使不实现，也应先明确它未来最小要做什么：

- 接收 `--name`
- 接收 `--output`
- 把当前仓库的最小内核模板复制到目标目录
- 生成目标项目自己的 `.env.example`、`app.yaml.example`
- 替换项目名、模块名、镜像名等占位符
- 生成后项目可独立执行 `make build`

### 推荐输入

- `--name`
- `--output`
- `--module-path` 可后置，不作为第一阶段必须项

### 推荐输出

在目标目录得到一份独立项目，例如：

```text
../my-saas/
  cmd/
  internal/
  db/
  migrations/
  docs/
  Makefile
  go.mod
  .env.example
  app.yaml.example
```

### 第一阶段明确不做

- 不做模板市场
- 不做远程下载模板
- 不做交互式问答向导
- 不做多种数据库/消息队列组合选择
- 不做动态插件安装

### 后续可扩展方向

- 支持不同项目模板：
  - `kernel-only`
  - `kernel+admin-api`
  - `kernel+job`
- 支持交互式初始化
- 支持公司内部模板仓库

## 命令二：new-module

### 职责

在当前项目中生成一个模块骨架。

例如：

```bash
go run ./cmd/cli new-module --name post
```

### 适用场景

- 已经在某个 SaaS 项目里
- 需要新起一个业务模块
- 想减少重复创建目录和空文件的工作

### 当前实现

当前仓库已经实现了一个最小版 `new-module`。

它会在当前项目根目录下固定生成：

```text
internal/domain/model/<name>.go
internal/domain/port/<name>.go
internal/app/usecase/<name>.go
internal/repo/pg/<name>_repo_pg.go
internal/delivery/http/handler/<name>_handler.go
db/query/<name>.sql
migrations/<timestamp>_add_<plural>.sql
```

### 设计原则

- 不允许自定义任意路径
- 路径由框架约定固定
- 生成器只负责稳定重复的骨架
- 具体 SQL、migration、usecase 逻辑由开发者手工补充

### 第一阶段能力边界

保留：

- `--name`
- `--migration-prefix`
- snake_case 名称校验
- CamelCase 实体名推导
- 简单复数推导
- 已存在文件拒绝覆盖

不做：

- 不自动注册 bootstrap
- 不自动加路由
- 不自动修改 `sqlc.yaml`
- 不自动执行 `sqlc generate`
- 不自动写 migration 内容
- 不自动写 Casbin 权限

### 后续可扩展方向

- `--with-test`
- `--with-route`
- `--with-query`
- `--with-migration`
- 自动生成模块 README
- 自动生成模块注册骨架

## 两个命令的边界

必须严格区分：

### new-project

- 生成“整个项目”
- 输出到指定目录
- 面向“从零开始”

### new-module

- 生成“当前项目里的一个模块”
- 输出到当前仓库固定结构
- 面向“项目增量开发”

## 推荐演进顺序

### 阶段 1

已完成：

- `new-module` 最小骨架生成

### 阶段 2

下一步建议：

- 给 `new-module` 补文档和 README 示例
- 明确生成后开发者还需要手工完成哪些步骤

### 阶段 3

再做：

- `new-project` 最小实现
- 基于当前仓库最小可运行内核复制到目标目录

### 阶段 4

最后再考虑：

- 模板变体
- 交互式初始化
- 可选模块预装

## 当前结论

当前仓库应把 CLI 明确定位为：

- `new-module`：当前已实现，服务“项目内模块生成”
- `new-project`：当前只定方案，后续实现，服务“新项目初始化”

这两者一起，才构成完整的 SaaS 脚手架体验。
