# OSS Optional Plan

This document defines the first infrastructure simplification step for the scaffold:

- object storage must become an optional capability
- the service must still start without MinIO or any other OSS provider
- file-related routes must not behave as if OSS were enabled when it is not configured

## Why This Change

Current startup does not hard-fail without OSS credentials, but the behavior is still misleading:

- the app falls back to `MockObjectStorage`
- file upload and download routes are still registered
- users can call file APIs even when real object storage is not enabled

That is not a real "optional dependency". It is only a startup fallback.

The scaffold should make this boundary explicit:

- database is required
- object storage is optional
- file capability depends on object storage being enabled

## Current State

Current code behavior:

- [`internal/bootstrap/di_repos.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/bootstrap/di_repos.go)
  - uses MinIO when `oss.endpoint/access_key/secret_key/bucket` are all present
  - otherwise falls back to `MockObjectStorage`
- [`internal/bootstrap/di_handlers_file.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/bootstrap/di_handlers_file.go)
  - always wires file handlers
- [`internal/bootstrap/di_routes.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/bootstrap/di_routes.go)
  - always registers file routes
- [`internal/app/usecase/file_storage.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/app/usecase/file_storage.go)
  - assumes storage is available once injected

Gap:

- startup is optional
- feature semantics are not optional

## Target State

After this change:

- the application starts normally without OSS configuration
- OSS-dependent file routes are not registered when OSS is disabled
- file-specific cron or cleanup jobs are not wired when OSS is disabled
- generated project docs clearly say OSS is optional
- users can still add MinIO later by filling config

## Scope

This phase should only solve OSS optionality.

In scope:

- bootstrap capability detection
- route registration gating
- file handler wiring gating
- config and README updates
- tests for enabled and disabled states

Out of scope:

- local filesystem storage adapter
- multiple OSS providers
- dynamic hot enable/disable
- refactoring file module into a full plugin system

## Recommended Design

Use explicit capability gating in bootstrap.

Recommended rule:

- OSS is enabled only when all required MinIO config fields are present
- when OSS is disabled:
  - do not inject `MockObjectStorage` into the runtime path
  - do not register file routes
  - do not expose fake upload or download URLs

Why not keep the current mock fallback:

- it hides misconfiguration
- it makes the scaffold look functional when file storage is actually unavailable
- it complicates later SQLite work by mixing "feature disabled" with "fake adapter enabled"

## Implementation Checklist

### 1. Add explicit OSS enable detection

Goal:

- centralize the rule "is OSS enabled"

Likely changes:

- [`internal/bootstrap/config.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/bootstrap/config.go)
- [`internal/bootstrap/di_repos.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/bootstrap/di_repos.go)

Checklist:

- add a helper such as `OSSConfig.Enabled() bool`
- treat `endpoint/access_key/secret_key/bucket` as required fields
- stop using `MockObjectStorage` in the normal app bootstrap path

### 2. Make file capability conditional

Goal:

- file module exists only when OSS is enabled

Likely changes:

- [`internal/bootstrap/di_handlers.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/bootstrap/di_handlers.go)
- [`internal/bootstrap/di_handlers_file.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/bootstrap/di_handlers_file.go)
- [`internal/bootstrap/di_routes.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/bootstrap/di_routes.go)

Checklist:

- do not wire `fileHandler` when OSS is disabled
- do not register `/file/*` routes when OSS is disabled
- keep startup healthy when file capability is absent

### 3. Keep domain behavior strict

Goal:

- no silent fallback, no fake success path

Likely changes:

- [`internal/app/usecase/file_storage.go`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/internal/app/usecase/file_storage.go)

Checklist:

- keep validation strict
- if file use cases are ever called without storage, return explicit validation or disabled-capability errors
- do not fabricate upload/download URLs

### 4. Update generated-project docs

Goal:

- reduce first-run confusion

Likely changes:

- [`README.md`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/README.md)
- [`README.zh-CN.md`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/README.zh-CN.md)
- [`app.yaml.example`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/app.yaml.example)
- [`.env.example`](/Users/luoshanjie/workspace/ai/base/clean_saas_starter/.env.example)

Checklist:

- document that OSS is optional
- explain that file APIs are available only when OSS is configured
- keep MinIO as the built-in adapter, not a mandatory dependency

### 5. Add tests

Goal:

- prove both enabled and disabled states work

Checklist:

- bootstrap test: app can initialize with empty OSS config
- route test: file routes are absent when OSS is disabled
- route test: file routes are present when OSS is enabled
- config test: `Enabled()` rule is deterministic

## Acceptance Criteria

This phase is complete only when all of the following are true:

- service can start with database configured and OSS empty
- `/api/v1/file/...` routes are not exposed when OSS is disabled
- service can still start and expose file routes when MinIO is configured
- README and config examples do not imply MinIO is mandatory
- no fake object storage URLs are returned in the disabled state

## Follow-up After OSS

Once OSS optionality is complete, the next phase is SQLite support.

That phase should use these constraints:

- PostgreSQL remains the primary production path
- SQLite is for local development, demos, and low-friction onboarding
- PostgreSQL RLS behavior cannot be assumed in SQLite mode
- tenant isolation in SQLite mode must be enforced in application code
