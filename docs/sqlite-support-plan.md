# SQLite Support Plan

This document defines the second infrastructure simplification step for the scaffold:

- PostgreSQL remains the primary production path
- SQLite is added as a low-friction local development and demo option
- MySQL is explicitly out of scope

## Why SQLite

The scaffold currently has two onboarding costs that are still too high for many users:

- a PostgreSQL instance is required
- users must understand PostgreSQL-specific schema and runtime behavior on day one

For a starter scaffold, that is unnecessary friction.

SQLite solves the right problem here:

- one file is enough to boot locally
- no extra service is required
- new users can try the scaffold immediately

SQLite is not meant to replace PostgreSQL in the main architecture.

## Non-Goals

This phase does not try to make all databases equal.

Explicitly out of scope:

- MySQL support
- full SQL portability across all engines
- keeping PostgreSQL RLS semantics in SQLite
- keeping one shared sqlc package for every database
- hot switching databases at runtime

## Current PostgreSQL Coupling

Current code is tightly coupled to PostgreSQL in four places:

### 1. Bootstrap and connection layer

- [`internal/bootstrap/db.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/bootstrap/db.go)
- [`internal/bootstrap/manual_di.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/bootstrap/manual_di.go)
- [`internal/bootstrap/di_repos.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/bootstrap/di_repos.go)

Current state:

- bootstrap expects `pgxpool.Pool`
- repo wiring is PostgreSQL-specific

### 2. Repository implementation layer

- [`internal/repo/pg/`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/repo/pg)
- [`internal/repo/pg/sqlcpg/`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/repo/pg/sqlcpg)

Current state:

- repo adapters use `pgx`
- generated code is PostgreSQL-only
- RLS setup is embedded in repo execution flow

### 3. SQL and schema layer

- [`sqlc.yaml`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/sqlc.yaml)
- [`db/query/*.sql`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/db/query)
- [`migrations/pgsql/0001_kernel_core.sql`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/migrations/pgsql/0001_kernel_core.sql)
- [`migrations/pgsql/demo_schema_init.sql`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/migrations/pgsql/demo_schema_init.sql)

Current state:

- `sqlc` is configured only for PostgreSQL
- schema uses PostgreSQL-only features such as:
  - `timestamptz`
  - `jsonb`
  - partial indexes
  - `current_setting(...)`
  - row-level security policies
  - `::uuid` / `::jsonb` / `::timestamptz`
  - `ILIKE`
  - `RETURNING`

### 4. Tenant isolation strategy

- [`internal/repo/pg/rls.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/repo/pg/rls.go)
- [`internal/delivery/http/middleware/rls_context.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/delivery/http/middleware/rls_context.go)

Current state:

- tenant isolation is strongly tied to PostgreSQL RLS

That cannot be preserved in SQLite.

## Target State

After this phase:

- the scaffold can run with PostgreSQL or SQLite
- PostgreSQL remains the default production-oriented path
- SQLite becomes the default low-friction local path
- tenant isolation in SQLite mode is enforced in application/repository code
- OSS remains optional, independent from database choice

## Security Boundary

PostgreSQL and SQLite do not provide the same isolation guarantees.

PostgreSQL mode:

- keeps the current RLS-based tenant isolation model
- adds a database-level protection layer below application code
- should remain the recommended production path

SQLite mode:

- does not have PostgreSQL-style RLS support
- cannot provide the same database-level isolation guarantee
- must rely on explicit tenant scoping in application and repository logic
- should be positioned as local development, demo, and onboarding infrastructure

This difference should be documented clearly in README and setup guides instead of being hidden behind a fake "database parity" story.

## Design Principles

### 1. Keep database choice explicit

Do not auto-detect from DSN shape only.

Recommended config:

- `DB_DRIVER=postgres`
- `DB_DRIVER=sqlite`

Keep `DB_DSN` for PostgreSQL.

For SQLite, add a dedicated path-like config:

- `SQLITE_PATH=./var/service.db`

This keeps intent clear and avoids overloading one DSN format for two engines.

### 2. Do not force PostgreSQL abstractions onto SQLite

Do not try to emulate:

- RLS
- `current_setting`
- PostgreSQL JSON operators
- PostgreSQL-specific casts

Instead:

- keep PostgreSQL repo path as-is
- add a separate SQLite repo path
- document the security difference explicitly

### 3. Start with the smallest useful kernel path

SQLite v1 only needs to support the core starter flow:

- tenant
- auth
- audit
- file upload session metadata
- RBAC policy storage

This is enough for local boot, login, tenant management, and basic kernel verification.

## Recommended Implementation Order

### Step 1. Add database driver config

Goal:

- choose PostgreSQL or SQLite at bootstrap time

Likely changes:

- [`internal/bootstrap/config.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/bootstrap/config.go)
- [`README.md`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/README.md)
- [`README.zh-CN.md`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/README.zh-CN.md)
- [`.env.example`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/.env.example)
- [`app.yaml.example`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/app.yaml.example)

Checklist:

- add `DB_DRIVER`
- add `SQLITE_PATH`
- default local example to SQLite
- keep PostgreSQL examples documented for production-style usage

### Step 2. Introduce a database-neutral bootstrap boundary

Goal:

- stop passing `pgxpool.Pool` through the whole composition root

Likely changes:

- [`internal/bootstrap/manual_di.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/bootstrap/manual_di.go)
- [`internal/bootstrap/db.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/bootstrap/db.go)
- [`internal/bootstrap/di_repos.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/bootstrap/di_repos.go)

Checklist:

- introduce a driver-aware bootstrap branch
- keep PostgreSQL repo wiring under one branch
- add SQLite repo wiring under another branch
- avoid leaking driver-specific types above bootstrap

### Step 3. Add SQLite schema and migration baseline

Goal:

- allow a clean local SQLite database to be initialized

Likely changes:

- add SQLite schema files under `migrations/sqlite/`
- keep PostgreSQL files under `migrations/pgsql/`

Current baseline files:

- [`migrations/sqlite/0001_kernel_core.sql`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/migrations/sqlite/0001_kernel_core.sql)
- [`migrations/sqlite/demo_schema_init.sql`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/migrations/sqlite/demo_schema_init.sql)

Checklist:

- define SQLite equivalents for kernel tables
- replace PostgreSQL-only types with SQLite-safe types
- remove RLS and policy DDL from SQLite schema
- replace PostgreSQL-specific checks with portable constraints where possible

Recommended direction:

- `TEXT` for UUID values
- `TEXT` or `INTEGER` timestamps depending on consistency preference
- `TEXT` for JSON payloads in v1

### Step 4. Add SQLite repository adapters

Goal:

- make core use cases run on SQLite

Recommended structure:

- [`internal/repo/sqlite/`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/repo/sqlite)

Checklist:

- implement SQLite versions of:
  - auth repo
  - tenant repo
  - audit repo
  - file upload session repo
- keep interfaces in `internal/domain/port` unchanged
- enforce tenant filtering explicitly in SQLite queries and repo methods

Important rule:

- SQLite mode must never assume PostgreSQL RLS is protecting the query

### Step 5. Decide sqlc strategy

This needs an explicit choice before coding too much.

Recommended v1 strategy:

- keep current PostgreSQL sqlc generation as-is
- add a separate SQLite sqlc config if SQLite queries are also generated
- do not try to share one query set across both engines in the first iteration

Reason:

- portability work would be larger than the actual starter value

Two acceptable v1 options:

Option A:

- PostgreSQL uses `sqlc`
- SQLite repos are handwritten

Option B:

- PostgreSQL uses `sqlcpg`
- SQLite uses a second generated package such as `sqlcsqlite`

Recommended first choice:

- Option A

It is simpler and keeps the MVP small.

### Step 6. Adjust tenant isolation semantics

Goal:

- preserve correctness even though security guarantees differ

PostgreSQL mode:

- keep current RLS-based isolation

SQLite mode:

- tenant filters must be explicit in repository logic
- tests must assert tenant scoping behavior directly

This difference must be documented clearly.

## Acceptance Criteria

SQLite support is complete only when all of the following are true:

- service can start with `DB_DRIVER=sqlite`
- a fresh local SQLite database can be initialized from committed schema files
- auth and tenant flows work on SQLite
- audit writes work on SQLite
- OSS remains optional in SQLite mode
- docs clearly explain the PostgreSQL vs SQLite difference
- PostgreSQL path still passes existing tests

## Risks

### Risk 1. False parity expectation

Users may assume SQLite and PostgreSQL are equivalent.

Mitigation:

- document clearly that SQLite is for local/dev/demo
- document that PostgreSQL remains the stronger production path
- document that only PostgreSQL provides the current database-level RLS layer

### Risk 2. Over-expanding SQL portability work

If we try to make every query portable, the MVP will stall.

Mitigation:

- keep PostgreSQL SQL and SQLite SQL separate in v1

### Risk 3. Hidden tenant-scope regressions

Removing RLS in SQLite increases the chance of missing tenant filters.

Mitigation:

- add repo-level tenant isolation tests for SQLite

## Recommended Next Cut

The next code change should be only Step 1:

- add `DB_DRIVER`
- add `SQLITE_PATH`
- update config templates
- update README quick start for local SQLite onboarding

Do not start by rewriting repositories first.
