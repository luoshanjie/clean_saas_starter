# Clean SaaS Starter

English | [简体中文](README.zh-CN.md)

An open-source multi-tenant SaaS starter for building new systems on top of a reusable kernel instead of starting from a vertical business project.

## 1. What It Is

This repository is a generic SaaS scaffold, not a concrete business application.

Its goal is to provide out-of-the-box kernel capabilities for new SaaS systems:

- multi-tenant context and isolation
- authentication and RBAC
- runtime config, logging, and middleware
- audit and file upload primitives
- project and module scaffolding CLI

Current core stack:

- Go
- Echo
- sqlc
- Casbin
- PostgreSQL
- MinIO (optional, only required when file storage is enabled)

Authentication policy:

- login second factor is optional at the framework level
- by default, `/api/v1/auth/login` can return tokens directly after account/password verification
- when `auth.login_second_factor_enabled=true`, login switches to `login -> verify` with OTP challenge
- second factor delivery can be implemented by users with SMS, email, TOTP, or other adapters

Database policy:

- PostgreSQL is the recommended production path
- PostgreSQL can provide an extra database-level tenant isolation layer through RLS
- SQLite is intended for local development, demos, and low-friction onboarding
- SQLite does not provide PostgreSQL-style database-level RLS protection
- tenant isolation in SQLite mode must rely on application and repository logic
- SQLite bootstrap wiring is being added incrementally; the current runtime path still boots with PostgreSQL

Object storage policy:

- the framework ships with a built-in MinIO adapter
- object storage is optional; the service can run without OSS
- cloud-vendor SDKs are not bundled into the main scaffold
- Alibaba Cloud, Tencent Cloud, Cloudflare R2, AWS S3, or other providers should be added by users in their own adapters

Current `oss` config fields:

- `endpoint`
- `access_key`
- `secret_key`
- `bucket`
- `public_base_url`
- `use_ssl`

## 2. How To Use It

### Create A New Project

Generate a new project from this starter:

```bash
go run ./cmd/cli new-project --name my-saas --output ../my-saas
```

Optional:

```bash
go run ./cmd/cli new-project --name my-saas --output ../my-saas --module-path github.com/acme/my-saas
```

The generated project will:

- copy the current starter into the target directory
- skip local-only files such as `.git`, `.env`, `app.yaml`, `build`, `logs`
- replace the default module name, binary name, command path, and example database name

### Start The Generated Project

Inside the generated project:

```bash
cp .env.example .env
make build
make dev
```

Before `make dev`, make sure:

1. your database has been created
2. for PostgreSQL, SQL in `migrations/pgsql/` has been executed
3. SQLite baseline files are available under `migrations/sqlite/`
4. `.env` or `app.yaml` points to the correct database
5. configure OSS only if you want to enable file upload and download routes
6. enable login second factor only if your project really needs it

Authentication config example:

```yaml
auth:
  login_second_factor_enabled: false
```

Or in `.env`:

```bash
AUTH_LOGIN_SECOND_FACTOR_ENABLED=false
```

### Generate A Module In The Current Project

```bash
go run ./cmd/cli new-module --name post
```

Optional:

```bash
go run ./cmd/cli new-module --name post --with-test
```

By default it generates:

```text
internal/domain/model/post.go
internal/domain/port/post.go
internal/app/usecase/post.go
internal/repo/pg/post_repo_pg.go
internal/delivery/http/handler/post_handler.go
db/query/post.sql
migrations/pgsql/<timestamp>_add_posts.sql
```

When `--with-test` is enabled, it also generates:

```text
internal/app/usecase/post_test.go
```

Notes:

- `--name` must be `snake_case`
- output paths are fixed by the framework convention
- existing files will not be overwritten

After generation, you still need to complete:

1. Write migration DDL in the generated migration file.
2. Write sqlc queries in the generated `db/query/<name>.sql`.
3. Implement the generated `repo / usecase / handler` skeletons.
4. Register the module manually in bootstrap and routes when the module is ready.

## 3. How It Is Organized

This project follows a clean architecture style with a kernel-first layout.

Main directories:

- `cmd/`
  - service entrypoint and scaffold CLI
- `internal/domain/`
  - domain models, errors, ports, auth context
- `internal/app/usecase/`
  - application use cases
- `internal/repo/`
  - infrastructure adapters such as PostgreSQL, Casbin, MinIO, cache, SMS
- `internal/delivery/http/`
  - Echo handlers, middleware, response envelopes
- `internal/bootstrap/`
  - composition root, config loading, DI, routes, app startup
- `db/query/`
  - sqlc query definitions
- `migrations/`
  - database migrations, including PostgreSQL and SQLite baselines
- `docs/`
  - design and scaffold planning documents

Important docs:

- [docs/kernel-capability-boundary.md](docs/kernel-capability-boundary.md)
  - kernel vs business-module boundary
- [docs/oss-optional-plan.md](docs/oss-optional-plan.md)
  - make object storage optional before SQLite support
- [docs/sqlite-support-plan.md](docs/sqlite-support-plan.md)
  - add SQLite as the low-friction local database path
- [docs/scaffolding-cli-plan.md](docs/scaffolding-cli-plan.md)
  - `new-project` and `new-module` CLI plan

This part is mainly for contributors or developers who want to understand or improve the scaffold itself.
