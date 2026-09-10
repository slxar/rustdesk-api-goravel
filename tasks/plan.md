# Implementation Plan: RustDesk API on Goravel and HTMX

## Overview

Modernize `lejianwen/rustdesk-api` without breaking its RustDesk-facing API or existing databases. Goravel v1.18.0 will own application startup, routing, middleware, views, testing, and graceful shutdown. Existing proven services and GORM models stay behind a small Gin-driver compatibility adapter while routes are covered by contract tests. A new self-hosted HTMX v2.0.10 admin replaces the removed/separate SPA; the removed RustDesk web client is not restored.

Compatibility targets:

- RustDesk client 1.4.9 (stable, 2026-07-06), plus additive current-master request fields and endpoints where they do not break stable clients.
- RustDesk server 1.1.16 (stable, 2026-07-20).
- Goravel 1.18.0 and its Gin driver, using Go 1.25 as required by the official scaffold.
- Existing SQLite, MySQL, and PostgreSQL data created by rustdesk-api v2.

## Architecture Decisions

- Preserve upstream Git history and business logic. Goravel owns the process and HTTP route registry; a narrow adapter unwraps Goravel's official Gin context to call existing Gin handlers. This is smaller and safer than rewriting 100+ observable endpoints at once.
- Define the contract before migration. A checked-in route manifest and request/response tests guard every RustDesk route; Swagger is documentation, not the source of truth.
- Serve the admin at `/_admin` with Go templates and self-hosted HTMX. Use semantic HTML, normal form actions as fallback, HTMX partial swaps, secure server-side sessions, CSRF protection, and no auth tokens in browser storage.
- Do not restore DMCA-removed web-client code or weaken E2EE. Logged-in secure connections require a RustDesk server that implements secure-TCP/API-token validation; official server 1.1.16 does not. This remains an explicit external integration gate.
- Keep existing database table names and data. Before schema mutation, back up SQLite, use WAL/full synchronization, detect unexpected missing persistent data, and provide a tested backup command. No dev or production test records will be created.
- Resolve actionable upstream issues in the API/UI and classify server/client/deployment questions honestly instead of claiming they are code fixes.

## Source Baseline

- Reference master: `c5687e150615bfb00d9d105f0883a2802750a8ad`.
- Reference release: `v2.7` / `222792419f8b55c17b36bf5f423ec5ed7a53e2c4`.
- GitHub issue snapshot: 433 issues, 106 open and 327 closed on 2026-08-10.
- Baseline `go test ./...` and `go build ./cmd/apimain.go` fail because upstream does not commit `go.sum`.

## Task List

### Phase 1: Contracts and Reproducibility

#### Task 1: Record upstream issue and compatibility contracts

**Description:** Check all upstream issues, classify the open set by owner/disposition, and record the stable/current RustDesk HTTP contract and the secure-TCP server boundary.

**Acceptance criteria:**

- [ ] The audit records all 433 issue numbers through a reproducible snapshot and explicitly triages every open issue category.
- [ ] The compatibility matrix covers auth, address books, peers/groups, heartbeat/sysinfo, audit, server config, `/audit/alarm`, and `/switch-grant`.
- [ ] Server/client-only and unsafe E2EE bypass proposals are marked out of API scope with evidence.

**Verification:**

- [ ] Snapshot counts match GitHub on the audit date.
- [ ] All source URLs point to official repositories, releases, docs, or issues.

**Dependencies:** None

**Files likely touched:** `docs/upstream-issue-audit.md`, `docs/rustdesk-compatibility.md`

**Estimated scope:** Medium

#### Task 2: Make the inherited tree reproducible

**Description:** Fix the root build generators, commit an honest `go.sum`, remove hard-coded LAN Redis test assumptions, and establish green pre-migration commands.

**Acceptance criteria:**

- [ ] `go build ./...` can build all non-generator packages.
- [ ] Redis tests use `TEST_REDIS_ADDR` and skip clearly when Redis is unavailable.
- [ ] No test writes to repository-owned runtime paths.

**Verification:**

- [ ] `go test ./...`
- [ ] `go build ./...`
- [ ] `go vet ./...`

**Dependencies:** None

**Files likely touched:** `generate_api.go`, `generate_run.go`, `lib/cache/*_test.go`, `go.sum`

**Estimated scope:** Medium

### Checkpoint: Contracts

- [ ] Route/issue/compatibility ledgers are reviewable.
- [ ] Inherited baseline is reproducible before framework changes.

### Phase 2: Goravel Foundation

#### Task 3: Boot the existing application with Goravel

**Description:** Add the minimal official Goravel v1.18 application lifecycle, config, facades, and graceful shutdown while retaining the existing CLI and initialization contract.

**Acceptance criteria:**

- [ ] `apimain` starts and stops through Goravel.
- [ ] Existing YAML/environment configuration continues to work.
- [ ] Go and Docker builds use Go 1.25 and pinned Goravel 1.18 modules.

**Verification:**

- [ ] Focused bootstrap tests pass.
- [ ] `go build ./...` and a signal-based smoke test pass.

**Dependencies:** Task 2

**Files likely touched:** `bootstrap/*`, `config/goravel_*`, `app/facades/*`, `cmd/apimain.go`, `go.mod`

**Estimated scope:** Medium

#### Task 4: Preserve every API route through a compatibility adapter

**Description:** Register `/api` and `/api/admin` with Goravel's Gin driver, adapt existing handlers/middleware once, and keep the old response semantics.

**Acceptance criteria:**

- [ ] Every inherited RustDesk and admin API method/path is registered.
- [ ] Auth middleware, rate limiting, fallback responses, static uploads, and request context values still work.
- [ ] New `/api/audit/alarm` and `/api/switch-grant` contracts are additive.

**Verification:**

- [ ] Route manifest test fails on any missing or changed method/path.
- [ ] Focused HTTP contract tests cover public, authenticated, validation, and error paths.

**Dependencies:** Task 3

**Files likely touched:** `app/http/legacy/*`, `routes/api.go`, `routes/admin_api.go`, `tests/feature/*`

**Estimated scope:** Medium

### Checkpoint: Goravel API

- [ ] RustDesk 1.4.9 API contract tests pass.
- [ ] Goravel owns the running HTTP application.

### Phase 3: Secure HTMX Admin

#### Task 5: Add admin session, CSRF, and authentication shell

**Description:** Build server-rendered login/logout and layout navigation using secure cookies/sessions and CSRF-protected forms.

**Acceptance criteria:**

- [ ] Password, LDAP, and configured OIDC entry points remain available.
- [ ] Login rotates session state; logout revokes the token and clears the cookie.
- [ ] Tokens never enter localStorage, URLs, or rendered logs.

**Verification:**

- [ ] Auth/session/CSRF feature tests pass.
- [ ] Cookie flags and security headers are asserted.

**Dependencies:** Task 4

**Files likely touched:** `app/http/controllers/admin/*`, `app/http/middleware/*`, `routes/admin.go`, `resources/views/admin/*`

**Estimated scope:** Medium

#### Task 6: Deliver responsive management workflows

**Description:** Add dashboard, users, peers/alias search, groups, address books/tags, audits, tokens, OAuth/LDAP configuration, and RustDesk server configuration as HTML and HTMX fragments.

**Acceptance criteria:**

- [ ] Core resources can be listed, searched, created/updated, and safely deleted where the inherited API permits.
- [ ] Destructive actions require confirmation and authorization; audit deletion can be disabled by policy.
- [ ] Empty/error/loading states and 320/768/1024/1440 px layouts are usable by keyboard.

**Verification:**

- [ ] Focused controller/template tests pass.
- [ ] Real-browser login, navigation, filter, form, and logout flows pass with no console errors or accessibility blockers.

**Dependencies:** Task 5

**Files likely touched:** `app/http/controllers/admin/*`, `resources/views/admin/*`, `public/css/admin.css`, `public/vendor/htmx/*`

**Estimated scope:** Large, implemented as independent resource slices

### Phase 4: Actionable Issue Fixes

#### Task 7: Harden data durability and operations

**Description:** Prevent silent SQLite replacement, enable durable SQLite pragmas, add atomic backups, document persistent volumes, and schedule optional audit retention.

**Acceptance criteria:**

- [ ] Startup refuses an unexpected empty replacement database unless explicitly acknowledged.
- [ ] SQLite uses WAL, busy timeout, foreign keys, and full synchronization.
- [ ] Backup and restore instructions are executable and retention never deletes logs by default.

**Verification:**

- [ ] Temporary-directory durability/backup tests pass without shared dev/prod data.
- [ ] Container restart smoke test preserves data.

**Dependencies:** Task 3

**Files likely touched:** `lib/orm/sqlite.go`, `cmd/*`, `config/*`, `docs/operations.md`

**Estimated scope:** Medium

#### Task 8: Resolve auth, filtering, and current-client deltas

**Description:** Implement the API-owned actionable issues: 64-character usernames, alias filtering, LDAP diagnostics/admission behavior, token expiry consistency, disabled-user revocation, optional TOTP, immutable-audit policy, current audit fields/alarm, and signed switch grants.

**Acceptance criteria:**

- [ ] Each implemented issue has a regression test and audit-ledger link.
- [ ] Boundary validation and authorization are consistent and backward compatible.
- [ ] No connection/key verification is bypassed.

**Verification:**

- [ ] Focused tests demonstrate RED before GREEN for each behavior.
- [ ] Full test/build/vet gates pass.

**Dependencies:** Tasks 4 and 5

**Files likely touched:** Existing request/service/model/controller files plus focused tests

**Estimated scope:** Large, implemented as independent issue slices

### Checkpoint: Complete Product

- [ ] API, HTMX admin, durability, and actionable issue tests are green.
- [ ] Remaining issues are explicitly assigned to API, RustDesk client/server, deployment, or unsupported web-client scope.

### Phase 5: Verification and Release

#### Task 9: Verify, review, and document the release

**Description:** Run full local/container/browser/security checks, refresh the code graph, and obtain an independent correctness/security/simplicity review.

**Acceptance criteria:**

- [ ] Tests, build, vet, dependency audit, diff check, browser flows, and container persistence checks pass.
- [ ] All critical/required review findings are fixed.
- [ ] README, migration, operations, compatibility, security, and changelog docs match the implementation.

**Verification:** See the release checklist in `tasks/todo.md`.

**Dependencies:** Tasks 1-8

**Files likely touched:** `README*`, `CHANGELOG.md`, `docs/*`, CI and container files

**Estimated scope:** Medium

#### Task 10: Publish the new repository

**Description:** Create a public repository under the authenticated GitHub account, push the reviewed branch/history and release tag, and verify remote files/actions.

**Acceptance criteria:**

- [ ] The new repository is public, retains upstream attribution/license, and has an `upstream` remote.
- [ ] Default branch and release tag point to the locally verified commit.
- [ ] GitHub reports the expected commit and CI configuration.

**Verification:**

- [ ] Compare local and remote commit SHAs.
- [ ] Read back repository visibility, default branch, and release URL through GitHub.

**Dependencies:** Task 9

**Files likely touched:** Git metadata and GitHub repository state only

**Estimated scope:** Small

## Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Official RustDesk server lacks logged-in secure-TCP token support | Logged-in clients may fail E2EE connection despite correct API | Never bypass verification; publish the exact server capability gate and test API compatibility independently |
| Existing databases vary across SQLite/MySQL/PostgreSQL versions | Migration could lose or reinterpret data | Preserve tables, back up first, test adopt-existing migrations, and make rollback explicit |
| 100+ observable routes have undocumented quirks | A framework rewrite can break clients | One adapter, a route manifest, golden response tests, and additive changes only |
| HTMX admin expands auth/CSRF surface | Account compromise or unauthorized mutations | HttpOnly/SameSite cookies, session rotation, CSRF middleware, authorization on every mutation, strict output escaping |
| Upstream issue list mixes code, support, server, client, and removed web-client requests | Impossible or unsafe promises | Record every issue and its real owner/disposition; implement only API-owned actionable work |
| Goravel v1.18 requires Go 1.25 while local Go is 1.24.3 | Local toolchain mismatch | Pin Go 1.25 in `go.mod`, CI, and Docker; allow Go toolchain auto-download during verification |

## Open Gates

- Physical RustDesk client 1.4.9 login/address-book replay against a compatible secure-TCP server.
- Live deployment, reverse proxy, TLS, and persistent-volume validation.
- Any RustDesk server fork or patch needed for logged-in connection authorization.
