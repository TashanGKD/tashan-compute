# Space, File, Snapshot, and Secret Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver personal and organization spaces with quota-safe MinIO storage, versioned files, resumable uploads, conflict-safe sync, immutable snapshots and encrypted run-time Secret references through matching API, CLI and Skill capabilities.

**Architecture:** PostgreSQL owns authorization, metadata, versions, reservations and snapshot manifests. MinIO stores immutable content-addressed blobs and temporary multipart uploads under server-generated keys; paths are display metadata only. Every write is authorized against `space_id`, quota is reserved transactionally, and every server capability has a real CLI command and Skill reference.

**Tech Stack:** Go 1.26, PostgreSQL 17, MinIO/S3, Redis 8, `minio-go/v7`, AES-256-GCM envelope encryption, Cobra, Docker Compose.

---

## Task 1: Space and file schema

**Files:**
- Create: `migrations/0002_spaces_files.sql`
- Create: `internal/space/model.go`
- Create: `internal/store/postgres/spaces.go`
- Test: `internal/store/postgres/spaces_test.go`

- [ ] Write a failing integration test proving account creation yields one 50 GiB personal space, organization creation yields one 500 GiB organization space, and duplicate creation cannot create a second canonical space.
- [ ] Run `go test ./internal/store/postgres -run TestCanonicalSpaces -v`; expect missing migration/repository failure.
- [ ] Add `spaces`, `directory_acls`, `file_entries`, `file_versions`, `upload_sessions`, `upload_parts`, `storage_reservations`, `trash_entries`, `workspace_snapshots`, `snapshot_files`, `secrets`, and `secret_grants`. Use UUID keys, foreign keys, CHECK constraints for space type/role/state, unique canonical owner/org constraints and bigint byte counters.
- [ ] Update account and organization transactions to create canonical spaces atomically. A failed space insert rolls back the account/organization.
- [ ] Run the schema tests twice and verify migration checksum protection still passes.
- [ ] Commit with `feat(space): add canonical space schema`.

## Task 2: Space authorization and quota ledger

**Files:**
- Create: `internal/space/authorize.go`
- Create: `internal/space/authorize_test.go`
- Create: `internal/store/postgres/quota.go`
- Test: `internal/store/postgres/quota_test.go`

- [ ] Write rejection tests for: another user's personal space, a removed organization member, viewer write/run, guessed space UUID, restricted directory prefix collision, and simultaneous reservations exceeding quota.
- [ ] Run targeted tests; expect missing authorization/quota functions.
- [ ] Implement `Authorize(principal, space, action, directoryACL)` with `read`, `write`, `run`, `admin` actions. Platform administrator does not inherit personal-space access.
- [ ] Implement `ReserveBytes` using `SELECT ... FOR UPDATE`; enforce `used_bytes + reserved_bytes + request <= quota_bytes`. Reservation release/commit is idempotent and scoped by space and upload.
- [ ] Run `go test -race ./internal/space ./internal/store/postgres -run 'Authorization|Quota' -v`.
- [ ] Commit with `security(space): enforce ACL and quota ledger`.

## Task 3: MinIO object adapter and isolated test service

**Files:**
- Modify: `compose.test.yml`
- Create: `internal/objectstore/store.go`
- Create: `internal/objectstore/minio.go`
- Test: `internal/objectstore/minio_test.go`
- Modify: `.env.example`

- [ ] Write tests that reject caller-supplied object keys, absolute/path-traversal keys, wrong-space prefixes, checksum mismatch and aborted multipart uploads that leave parts behind.
- [ ] Start MinIO only in the test Compose project and run the tests; expect missing adapter.
- [ ] Implement server-generated keys `spaces/<space-id>/blobs/<sha256-prefix>/<sha256>` and `temporary/<upload-id>/...`. The adapter accepts typed IDs and digests, never raw paths.
- [ ] Implement multipart create/upload-part/complete/abort and `Stat/Get/Delete`; complete verifies aggregate SHA-256 before publishing immutable content.
- [ ] Verify an aborted or failed upload has no temporary parts and no published blob.
- [ ] Commit with `feat(storage): add isolated MinIO adapter`.

## Task 4: Resumable upload state machine

**Files:**
- Create: `internal/file/upload.go`
- Create: `internal/file/upload_test.go`
- Create: `internal/store/postgres/uploads.go`
- Test: `internal/store/postgres/uploads_test.go`

- [ ] Write state-machine tests for duplicate part number with different checksum, part larger than declared, wrong total checksum, expired upload, concurrent completion, client disconnect, quota release on failure and idempotent retry.
- [ ] Run targeted tests; expect missing upload service.
- [ ] Implement states `initiated`, `uploading`, `completing`, `completed`, `aborted`, `expired`. Only the owning account/device and authorized space writers may mutate an upload.
- [ ] Persist part ETag, SHA-256 and size. Completion locks upload and reservation rows, verifies declared totals, completes MinIO multipart, creates file/version and converts reservation to used bytes in one recoverable workflow.
- [ ] Add compensating cleanup for object-success/database-failure and database-success/object-timeout by querying actual object state before retry.
- [ ] Commit with `feat(file): add resumable upload lifecycle`.

## Task 5: Versioned files, trash and reads

**Files:**
- Create: `internal/file/service.go`
- Create: `internal/file/service_test.go`
- Create: `internal/store/postgres/files.go`
- Test: `internal/store/postgres/files_test.go`

- [ ] Write rejection tests for `../`, absolute paths, NUL, encoded separators, Unicode normalization collisions, guessed file/version IDs, cross-space version reads and permanent delete without explicit confirmation.
- [ ] Run tests; expect missing file service.
- [ ] Normalize display paths to UTF-8 NFC with `/` separators and reject empty segments, dot segments and control characters. Authorization remains object/space based.
- [ ] Implement list/read/download metadata, version history, create-new-version, trash, restore and permanent purge. Trash continues counting toward quota; purging decrements used bytes only when the last reference to a blob disappears.
- [ ] Stream downloads with bounded buffers and checksum/ETag headers; never load whole files in API memory.
- [ ] Commit with `feat(file): add versioned file operations`.

## Task 6: Conflict-safe `file sync`

**Files:**
- Create: `internal/sync/plan.go`
- Create: `internal/sync/plan_test.go`
- Create: `internal/cli/file_sync.go`
- Test: `internal/cli/file_sync_test.go`

- [ ] Write tests for changed local/remote SHA, deleted-then-recreated path, duplicate local case-folded names, symlink/hardlink input, nested `.git`, empty directory and remote modification after plan creation.
- [ ] Run tests; expect missing planner and CLI.
- [ ] Implement manifest comparison using path, base version ID, SHA-256, size and operation. Conflicts are explicit records; no last-writer-wins fallback.
- [ ] `tcompute file sync` without `--apply` prints one stable JSON plan and performs no upload. `--apply` requires the plan ID; overwrite additionally requires `--overwrite --yes` and an idempotency key.
- [ ] Revalidate remote base versions transactionally at apply time.
- [ ] Commit with `feat(sync): add conflict-safe file sync`.

## Task 7: Immutable workspace snapshots

**Files:**
- Create: `internal/snapshot/service.go`
- Create: `internal/snapshot/service_test.go`
- Create: `internal/store/postgres/snapshots.go`
- Test: `internal/store/postgres/snapshots_test.go`

- [ ] Write tests proving later file edits cannot alter a snapshot; snapshot roots are deterministic; cross-space versions, trashed versions and unauthorized Secret references are rejected.
- [ ] Run tests; expect missing snapshot service.
- [ ] Lock the visible file-version set, sort by normalized path, compute a Merkle-style SHA-256 root, and persist immutable `snapshot_files` rows. Database triggers reject update/delete of completed snapshots.
- [ ] Add `snapshot create|list|show` API and CLI with mandatory `--space`.
- [ ] Commit with `feat(snapshot): add immutable workspace inputs`.

## Task 8: Encrypted Secret store

**Files:**
- Create: `internal/secret/service.go`
- Create: `internal/secret/service_test.go`
- Create: `internal/secret/envelope.go`
- Create: `internal/store/postgres/secrets.go`
- Test: `internal/store/postgres/secrets_test.go`

- [ ] Write tests for CLI argv Secret rejection, wrong master key, ciphertext tamper, cross-space grant, viewer grant, revoked grant, read-after-create and logs/audit containing plaintext.
- [ ] Run tests; expect missing Secret service.
- [ ] Encrypt each Secret with a random data key using AES-256-GCM and bind AAD to `space_id`, Secret ID and version. Wrap data keys with the server master key; store ciphertext/nonces only. There is no plaintext read API.
- [ ] Implement set/list/grant/revoke and short-lived executor lease retrieval. Zero buffers after use where practical; never serialize plaintext into audit or idempotency bodies.
- [ ] Add CLI hidden input and stdin behavior identical to password safety.
- [ ] Commit with `security(secret): add encrypted scoped secrets`.

## Task 9: API, CLI, Skill and capability expansion

**Files:**
- Modify: `capabilities/manifest.json`
- Modify: `internal/cli/bindings.json`
- Modify: `skill/tashan-compute/capability-references.json`
- Create: `internal/httpapi/space_file_handlers.go`
- Create: `internal/cli/space_file_commands.go`
- Test: `internal/httpapi/space_file_routes_test.go`
- Test: `internal/cli/space_file_commands_test.go`

- [ ] Add capabilities for space list/show/quota; file list/read/upload/download/versions/trash/restore/purge/sync; snapshot create/list/show; Secret set/list/grant/revoke.
- [ ] Before handlers, run capability coverage and command/route tests; expect failures listing every missing surface.
- [ ] Implement handlers and commands from shared route constants and typed DTOs. Every operation requires explicit space ID; organization-space mutations also carry organization context validated server-side.
- [ ] Add same-named negative gate fixtures deleting one CLI command and one route, proving CI fails.
- [ ] Update Skill references and focused authentication/security references without duplicating API syntax.
- [ ] Commit with `feat(platform): expose complete space capabilities`.

## Task 10: Two-user space E2E and recovery gates

**Files:**
- Create: `tests/e2e/space_file_lifecycle_test.go`
- Create: `scripts/verify-space.sh`
- Create: `scripts/verify-space.self-test.sh`
- Modify: `.github/workflows/ci.yml`
- Modify: `README.md`
- Create: `docs/verification/2026-08-29-space-file.md`

- [ ] E2E: create two users and an organization; verify personal spaces remain private; add both as developers; upload the same project; force a sync conflict; resolve to a new version; create a snapshot; store/grant/revoke a Secret; trash/restore/download and verify checksums.
- [ ] Failure E2E: interrupt multipart upload, corrupt checksum, race quota reservations, remove a member during upload and restart API/MinIO/PostgreSQL before reconciliation.
- [ ] Write aggregator self-test that removes one required gate and proves `verify-space.sh --self-check` fails.
- [ ] Run `verify-foundation.sh`, `verify-space.sh`, race detector and MinIO leak inventory; require zero temporary uploads/reservations after tests.
- [ ] Update README only with verified commands and record exact evidence/limitations.
- [ ] Commit with `ci(space): verify multi-user file lifecycle`.

## Self-review

- Every space/file/Secret write starts with rejection tests and cleanup assertions.
- PostgreSQL is authorization/quota truth; MinIO keys never derive from user paths.
- `file sync` remains dry-run by default and cannot silently overwrite.
- Snapshots are immutable and reference exact file versions.
- Server capabilities, real HTTP routes, real CLI commands and Skill references expand together.
- Compute execution is deliberately excluded from this plan and begins only after `verify-space.sh` passes.
