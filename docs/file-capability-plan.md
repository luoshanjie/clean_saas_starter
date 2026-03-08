# File Capability Plan

## Why This Belongs In The Framework

File upload and download are not mandatory for every SaaS system, but they are common enough that the framework should provide them as an optional capability instead of forcing each project to rebuild them.

The framework should own the default safety rules because two hidden problems appear repeatedly when teams implement this themselves:

1. orphaned objects in object storage
2. unsafe download exposure such as leaked permanent URLs or weak access control

So the correct boundary is:

- file capability belongs in the framework
- object storage provider choice does not belong in the framework core
- file capability must be optional

## Current Implementation Review

The current code already contains a useful kernel-shaped file flow:

1. create upload session
2. presign upload URL
3. persist upload session metadata
4. confirm upload
5. presign download URL
6. clean expired unconfirmed uploads

Relevant files:

- [`internal/app/usecase/file_storage.go`](../internal/app/usecase/file_storage.go)
- [`internal/delivery/http/handler/file_handler.go`](../internal/delivery/http/handler/file_handler.go)
- [`internal/domain/port/object_storage.go`](../internal/domain/port/object_storage.go)
- [`internal/domain/port/file_upload_session_repo.go`](../internal/domain/port/file_upload_session_repo.go)
- [`internal/repo/storage/minio_object_storage.go`](../internal/repo/storage/minio_object_storage.go)
- [`internal/repo/pg/file_upload_session_repo_pg.go`](../internal/repo/pg/file_upload_session_repo_pg.go)
- [`internal/repo/sqlite/file_upload_session_repo_sqlite.go`](../internal/repo/sqlite/file_upload_session_repo_sqlite.go)

## What Is Good And Worth Reusing

These parts are worth keeping:

- presigned upload instead of proxy-upload through the application
- upload session table with `pending_confirm -> confirmed -> cleaned`
- cleanup job for expired pending uploads
- object storage abstraction through `port.ObjectStorage`
- provider-neutral runtime wiring
- file routes already gated by OSS enablement

This is a good foundation for an optional framework capability.

## Gaps In The Current Design

The current implementation is useful, but not yet a complete framework contract.

### 1. `files` table is not part of the active flow

The project still has a `files` table and `FileRepo`, but the current upload-confirm path does not write to it.

That means the system currently manages upload sessions, but not a stable file asset record.

Current outcome:

- upload session exists
- confirmed upload exists
- but there is no canonical file record for later ownership binding, deletion, or audit

### 2. confirm step does not verify the uploaded object

`UploadConfirm` currently trusts the client-side confirmation request.

It does not verify:

- object existence
- object size
- checksum
- mime type

For an MVP this is acceptable, but for a reusable framework this is a weak point.

### 3. download authorization is too thin

`DownloadPresign` currently accepts `file_url` and presigns it.

That is only safe if the route-level caller is already trusted enough. As a framework default, this is too weak because:

- there is no canonical file ownership check
- there is no binding to a business object
- there is no policy check beyond normal authenticated access

### 4. deletion consistency is not closed yet

The current code has cleanup for expired pending uploads, which is good.

But there is still no complete framework rule for confirmed files when a business record is deleted.

This is the important missing part for avoiding long-lived orphaned objects.

### 5. file semantics are still transport-oriented

The current API is centered around:

- upload session
- file URL

For framework-level reuse, the more stable center should become:

- file asset identity
- owner binding
- lifecycle state

## Recommended Framework Boundary

The framework should provide these default file capabilities.

### Kernel Default, Optional To Enable

- upload session creation
- upload confirmation
- download presign
- expired pending upload cleanup
- object storage adapter contract
- file asset metadata model
- file deletion consistency contract

### Business Module Responsibility

- what a file belongs to
- whether the file is avatar, attachment, video, document, etc.
- what constraints apply to that file type
- when a confirmed file becomes bound to a business entity
- who can download or delete a specific business-owned file

## Recommended Evolution Path

Do not rewrite from scratch.

Keep the current flow and evolve it in place.

### Step 1. Freeze The Existing Session Flow

Keep:

- `UploadSessionCreate`
- `UploadConfirm`
- `DownloadPresign`
- `CleanupExpiredUploadSessions`

These are already framework-shaped.

### Step 2. Introduce A Canonical File Asset Record

Use the existing `files` table properly instead of leaving it idle.

Recommended change:

- upload confirm should create a file asset record
- response should return `file_id` as the stable identifier
- later operations should prefer `file_id` over raw `file_url`

### Step 3. Separate Temporary Upload From Bound File

Clarify two phases:

- temporary uploaded object
- bound business file asset

This allows:

- cleanup of abandoned uploads
- safe ownership transition
- later business binding

### Step 4. Strengthen Download Policy

The framework should move from:

- `presign by file_url`

to:

- `presign by file_id`

Then business modules can apply ownership or permission checks before download.

### Step 5. Add Controlled Delete Contract

The framework should provide one default rule:

- if file record deletion requires object deletion, object deletion failure must block or enter explicit compensation

That matches the repository-wide consistency rule already defined in `AGENTS.md`.

## Recommended Decision

Use the current implementation as the base and refactor it.

Do not rewrite it from zero.

Reason:

- the current upload session workflow is already correct enough for a kernel MVP
- it already solves the most valuable infrastructure concerns
- rewriting now would mostly duplicate the same shape with more churn

The better move is:

- keep the current session flow
- connect it to a real file asset record
- tighten authorization and deletion semantics

## Immediate Next Slice

The next practical slice should be:

1. make `files` become the canonical confirmed file record
2. change download presign input from `file_url` toward `file_id`
3. define a framework-level delete contract for confirmed files

This keeps the current codebase moving forward without losing the useful work that already exists.
