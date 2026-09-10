# Official API parity audit — 2026-09-10

The successful RustDesk 1.4.9 inbound desktop test establishes the connection
path. It does not establish complete RustDesk Server Pro API parity.

## Authoritative references

- Released client: `rustdesk/rustdesk` tag `1.4.9`.
- Official Server Pro release: `1.8.6` (a separate product/version line from
  open-source `rustdesk-server` `1.1.16`).
- [API-token scopes](https://rustdesk.com/docs/en/self-host/rustdesk-server-pro/console/#api-token).
- [Official management scripts](https://github.com/rustdesk/rustdesk/tree/14a5ed45d95d7b31840fabc8a343b1ac221b2c65/res),
  pinned to the audited client-repository commit. These describe methods, request
  fields, filters, pagination, and successful response consumption.

No complete official public OpenAPI schema was found in the official documentation,
client or public Server Pro repository trees. The SCTG embedded API schema is a
third-party implementation and is not an official conformance reference.

## Current gaps

| Surface | Missing or different behavior |
|---|---|
| API token authorization | Local login tokens have no Pro resource-specific read/read-write scopes. |
| Users | Existing GET `/api/users` serves client visibility; official administrative create/invite/enable/disable/delete, force logout, and TOTP/verification management are separate contracts. |
| Devices | Administrative `/api/devices` lists, enable/disable/assign/delete, and client `/api/devices/cli` and `/api/devices/deploy` are absent. |
| User/device groups | Pro `/api/user-groups` and `/api/device-groups` CRUD and membership APIs are absent. |
| Address books | Official-script GET reads and native POST reads now share authenticated handlers with pagination. Shared-book/rule administration and full Pro authorization semantics still differ. |
| Strategies | Pro strategy lists, details, status and assignment APIs are absent. |
| Audit reading | Pro GET `/api/audits/{conn,file,alarm,console}` differs from the local ingest/admin APIs. |
| Admin/control roles | Official role definitions, assignments, enabled-state and membership APIs are absent. |
| Connection audit notes | Client active-audit GUID lookup and note update are absent. |

The public scripts establish requests to implement, but do not fully specify
policy precedence, all role/strategy definitions, token creation, failure responses,
or runtime enforcement. Exact conformance requires a versioned official reference
instance or a complete authoritative contract. Do not replace that evidence with
accept-all tokens, unverified policy behavior, or additional route registrations.

## Confirmed client audit correction

The released client's `post_conn_audit` serializes `LoginRequest.session_id` as
an unsigned 64-bit integer. The API previously decoded it as `float64` and rounded
values above 2^53 before storing them. `AuditConnForm.SessionId` now decodes directly
to `uint64`, then uses decimal formatting for the existing string database column.
Tests cover adjacent values above 2^53, maximum uint64, and invalid ranges/fractions.
No database migration or rewrite is required. Historical rounded values cannot be
reconstructed from their stored rounded representation alone.

## Address-book read correction

The official `res/ab.py` uses GET for personal books, shared profiles, peers and
tags. RustDesk 1.4.9 `flutter/lib/models/ab_model.dart` uses POST for those reads
and paginates shared profiles and peers using `current` and `pageSize=100`.
Both methods now work behind the existing account-token authentication and
book-level permissions. Shared profiles return the actual total, unique books,
maximum effective permission and deterministic pages, including books beyond the
previous 100-owned-book cutoff. Peers honor pagination beyond the previous 1,000
row cutoff and accept parameterized SQL-LIKE `id`/`alias` patterns used by the
script. Empty pages are arrays. Shared `name` filtering is exact; the public
script's exact-name lookup works, but broader official name-matching semantics
remain unverified.

`http/router/addressbook_read_test.go` exercises authenticated GET and POST with
105 owned books, duplicate/group rules, inaccessible and orphaned books, 1,050
peers, filtered pages, invalid pagination and unauthorized reads. Test records
exist only in an in-memory SQLite database. Omitting pagination retains a
1,000-item default; explicit page sizes are limited to 1,000.

The API Swagger artifacts can be regenerated with:

```sh
go run github.com/swaggo/swag/cmd/swag@v1.16.3 init -g cmd/apimain.go --output docs/api --instanceName api --exclude http/controller/admin --parseDependencyLevel 1
```
