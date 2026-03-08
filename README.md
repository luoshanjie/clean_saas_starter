# Clean SaaS Starter

An open-source multi-tenant SaaS starter with clean architecture, built on Echo, sqlc, and Casbin.

## Scope

This repository is positioned as a reusable SaaS kernel, not a vertical business application.

Built-in kernel capabilities currently include:

- multi-tenant context and isolation
- authentication and RBAC
- runtime config, logging, middleware
- audit and file upload primitives

## Object Storage Policy

The framework only ships with a built-in MinIO adapter.

It does not include cloud-vendor object storage SDKs by default. Support for Alibaba Cloud, Tencent Cloud, Cloudflare R2, AWS S3, or other providers should be added by framework users in their own adapters.

Current `oss` config is designed around the built-in MinIO adapter:

- `endpoint`
- `access_key`
- `secret_key`
- `bucket`
- `public_base_url`
- `use_ssl`

## CLI

Current scaffold CLI supports module skeleton generation inside the current project:

```bash
go run ./cmd/cli new-module --name post
```

Optional:

```bash
go run ./cmd/cli new-module --name post --with-test
```

By default it generates these files under the current repository root:

```text
internal/domain/model/post.go
internal/domain/port/post.go
internal/app/usecase/post.go
internal/repo/pg/post_repo_pg.go
internal/delivery/http/handler/post_handler.go
db/query/post.sql
migrations/<timestamp>_add_posts.sql
```

When `--with-test` is enabled, it also generates:

```text
internal/app/usecase/post_test.go
```

Notes:

- `--name` must be `snake_case`
- `--with-test` adds a minimal usecase test skeleton
- output paths are fixed by the framework convention
- existing files will not be overwritten

After generation, you still need to complete:

1. Write migration DDL in the generated migration file.
2. Write sqlc queries in the generated `db/query/<name>.sql`.
3. Implement the generated `repo / usecase / handler` skeletons.
4. Register the module manually in bootstrap and routes when the module is ready.

## Docs

- See [kernel-capability-boundary.md](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/docs/kernel-capability-boundary.md) for the kernel vs module boundary.
- See [scaffolding-cli-plan.md](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/docs/scaffolding-cli-plan.md) for the `new-project` and `new-module` CLI plan.
