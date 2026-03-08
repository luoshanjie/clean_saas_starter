# SaaS Kernel Capability Boundary

本文档用于定义本仓库作为“可插拔多租户 SaaS 内核 + 业务模块”脚手架时的能力边界，作为后续删除旧业务代码、整理目录结构、拆分模块的基线。

## 目标

本仓库的目标不是承载某个具体行业业务，而是为新建 SaaS 系统提供开箱即用的基础能力：

- 多租户
- 认证与授权
- 可扩展模块装配
- 配置、日志、错误处理
- 异步任务与事件
- 基础设施适配器

判断原则：

- 凡是跨 SaaS 场景稳定复用的能力，进入内核。
- 凡是带明显行业语义、生命周期和权限模型不通用的能力，进入业务模块。
- 凡是尚未稳定、但显然属于通用基础设施的能力，先放入“可选内核能力”，不要混入业务模块。

## 必须内核化

以下能力默认由框架提供，属于第一天就应该具备的 SaaS 基础能力。

### 1. Tenant

职责：

- 租户模型
- 租户启用/禁用状态
- 平台级与租户级作用域区分
- Tenant Context 注入与传播
- 数据隔离边界定义

最小能力：

- 平台用户可无 `tenant_id`
- 租户用户必须绑定 `tenant_id`
- 请求链路中可读取当前 `tenant_id`、`user_id`、`scope_type`
- 数据库访问层支持基于上下文的隔离控制

### 2. IAM

职责：

- 用户基础模型
- 凭证管理
- 登录、刷新、登出
- 密码策略
- Token 版本控制
- 基础会话失效能力

建议默认方案：

- `access token + refresh token`
- `access token` 短时效
- `refresh token` 可轮换
- 会话失效基于 `token_version` 或持久化 session 记录

### 3. Authz

职责：

- 用户-角色-权限三段式模型
- 平台级 / 租户级权限隔离
- 权限检查接口
- 权限命名空间约束

边界说明：

- 内核只提供通用 RBAC 能力
- Casbin 是默认实现适配器，不是业务层依赖对象
- 业务模块只能声明自己的权限，不直接耦合 Casbin 细节

### 4. Runtime

职责：

- 配置加载
- 环境变量覆盖
- 应用生命周期管理
- 日志
- 错误码与错误映射
- HTTP 中间件

最小能力：

- 结构化日志
- request id
- recover
- 认证中间件
- Tenant Context 中间件
- 权限校验中间件
- 限流中间件

### 5. Module

职责：

- 模块注册
- 模块路由挂载
- 模块依赖注入边界
- 模块权限声明
- 模块迁移归属约定

约束：

- 第一阶段采用编译期静态注册，不做动态热插拔
- 模块只能依赖内核公开接口
- 内核不能反向依赖业务模块

### 6. Job

职责：

- 异步任务模型
- 任务状态流转
- 重试机制
- 幂等控制
- 定时任务基础能力

设计方向：

- 先定义 `JobDispatcher`、`JobStore`、`JobHandler`
- 第一阶段可先用数据库实现
- 不在第一阶段绑定具体 MQ 产品

### 7. Event

职责：

- 领域事件 / 应用事件发布接口
- 订阅与处理器注册
- 异步任务与事件协作边界

目标：

- 为 AI 内容生成、转码、外部回调、数据同步等异步流程提供统一扩展点

### 8. Infra Ports

职责：

- DB
- Cache
- Object Storage
- SMS
- Email
- Webhook

约束：

- 业务层依赖端口接口
- 第三方 SDK 只能出现在基础设施适配器层
- 框架主干不内置多云厂商 SDK；对象存储默认只提供 MinIO 适配器
- 其他对象存储实现（如阿里云、腾讯云、Cloudflare R2、AWS S3）由使用者自行添加

## 可选内核能力

这些能力通常属于 SaaS 通用能力，但不要求第一阶段全部落地。应作为内核扩展项，不应混入业务模块。

### 1. SSO

- OIDC
- SAML
- 企业微信 / 飞书 / 钉钉等企业身份源接入

### 2. 多端登录控制

- 单设备登录
- 设备数限制
- 主动踢下线
- 风险会话失效

### 3. 审计日志

- 操作日志接口
- 审计事件记录
- 审计查询默认实现

### 4. Feature Flag / Tenant Feature

- 按租户启用模块
- 按租户控制功能开关
- 灰度发布支持

### 5. Scheduler

- 周期任务调度
- 失败重试与补偿
- 清理类后台任务

## 必须模块化

以下能力必须作为业务模块存在，不应进入内核。

### 1. 行业实体

例如：

- 客户
- 订单
- 工单
- 商品目录
- 媒体资源
- 内容集合
- AI 任务记录

判断标准：

- 业务名词不能跨行业稳定复用
- 生命周期和状态机不属于 SaaS 通用能力

### 2. 行业流程

例如：

- 客户审批
- 订单流转
- 商品导入
- 媒体内容管理
- AI 生成 / 抽取 / 转码流程定义

说明：

- 底层任务框架属于内核
- 具体任务定义属于模块

### 3. 行业权限

例如：

- `tenant.user.manage`
- `tenant.ticket.manage`
- `platform.media.manage`

说明：

- 权限引擎属于内核
- 权限命名空间与策略内容属于模块

### 4. 行业接口与报表

例如：

- 业务统计看板
- 内容资源管理
- 外部身份绑定流程
- 运营活动页面

## 当前仓库代码归类

以下归类用于指导后续删除和整理。

### 可保留并收敛为内核的部分

- `internal/domain/authctx`
- `internal/domain/errors`
- `internal/domain/port/cache.go`
- `internal/domain/port/object_storage.go`
- `internal/domain/port/permission.go`
- `internal/domain/port/sms_sender.go`
- `internal/repo/pg/rls.go`
- `internal/repo/casbin`
- `internal/repo/cache`
- `internal/repo/storage`
- `internal/bootstrap/config.go`
- `internal/bootstrap/http.go`
- `internal/delivery/http/middleware`
- 与用户、租户、认证、权限直接相关的模型 / repo / usecase / handler

### 需要重命名或收敛边界后保留的部分

- `internal/bootstrap/manual_di.go`
- `internal/bootstrap/di_*`
- `cmd/service`
- `db/query` 中与租户、用户、认证、RBAC 相关的 SQL

说明：

- 这些文件的职责是对的
- 但命名、装配方式和模块边界仍然带有当前项目历史痕迹

### 应视为旧业务模块并逐步移出的部分

- `crm_account`
- `ticketing`
- `catalog`
- `media_library`
- `content_collection`
- `workflow_approval`
- `analytics_dashboard`
- `campaign_ops`

涉及目录：

- `internal/domain/model`
- `internal/domain/port`
- `internal/app/usecase`
- `internal/repo/pg`
- `internal/delivery/http/handler`
- `db/query`
- `migrations`

说明：

- 这些能力不是内核
- 后续要么删除，要么迁移为示例业务模块

## 第一阶段拆分顺序

为了遵循 KISS 和最小改动原则，建议按以下顺序整理。

### 阶段 1：冻结内核最小范围

保留并整理：

- tenant
- user
- credential
- auth
- rbac
- runtime
- middleware
- infra ports

输出物：

- 内核目录边界
- 默认模块注册接口
- 保留能力清单

### 阶段 2：识别并隔离旧业务代码

处理方式：

- 标记哪些目录属于教学业务
- 从 bootstrap 路由装配中拆出
- 从默认启动流程中移除

目标：

- 让主干项目先能作为“纯内核”启动

### 阶段 3：引入模块注册机制

最小实现：

- `Module` 接口
- 模块路由注册
- 模块权限声明
- 模块迁移声明

目标：

- 让旧业务能力未来可以作为示例模块重新接回，而不是继续粘在内核里

### 阶段 4：补 Job / Event 基础设施

目标：

- 为 AI 应用和异步流程预留稳定扩展点

约束：

- 先做最小接口与默认实现
- 不提前引入复杂消息系统

## 当前明确不做的事情

- 不在第一阶段引入 ABAC / ReBAC
- 不在第一阶段做动态热插拔插件系统
- 不在第一阶段为未来行业预设抽象层次
- 不在第一阶段绑定具体 MQ、工作流引擎或分布式调度器

## 文档用途

本文件作为后续重构的判断依据：

- 删除旧业务代码时，以本文件判定是否属于内核
- 新增默认能力时，以本文件判定是否应进入内核
- 设计新模块时，以本文件判定其依赖方向是否正确
