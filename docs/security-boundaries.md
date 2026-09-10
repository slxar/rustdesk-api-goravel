# Security boundaries

This is the release threat model for the Goravel/HTMX migration. It records
controls that exist in the code and the remaining deployment gates.

## Assets and trust boundaries

| Boundary | Untrusted input | Protected assets | Controls |
|---|---|---|---|
| RustDesk client to `/api` | Headers, JSON, device identifiers, audit data | Access tokens, account/address-book data, audit database | Existing RustAuth, token expiry and enabled-user checks, Gin/Goravel body limits, binding validation, parameterized GORM queries |
| Browser to `/_admin` | Credentials, forms, query filters, rendered database text | Administrator authority, session token, configuration metadata | Server-side session, Secure/HttpOnly/SameSite=Lax cookie scoped to `/_admin`, session rotation/invalidation, CSRF, `IsAdmin`, login limiter and captcha, Go template escaping |
| Application to database/cache | Configuration and stored records | Entire persisted service state | SQLite WAL/FULL/foreign keys/busy timeout, atomic snapshot command, parameterized queries, documented durable volume |
| Application to LDAP/OIDC | Remote identity-provider responses | Account mapping and login authority | Existing provider validation and inherited service logic; TLS and provider availability remain deployment gates |
| RustDesk client to hbbs | Server key, access token, rendezvous responses | E2EE/server identity | Verification is owned by RustDesk client/server; this API never bypasses key or signature checks |

## STRIDE disposition

- **Spoofing:** protected admin pages require an enabled administrator token in
  a server-side session. Password failures use the existing limiter/captcha.
  RustDesk alarm posts are necessarily unauthenticated and are not treated as
  proof of identity.
- **Tampering:** CSRF protects state-changing admin forms; output is escaped;
  filters are bound parameters; SQLite backups are written to a sibling
  temporary file and atomically renamed.
- **Repudiation:** inherited login, connection and file audits remain. Alarm
  events record the exact client payload, but their unauthenticated provenance
  is an explicit limitation.
- **Information disclosure:** the HTMX UI does not render RustDesk private
  keys, OAuth client secrets, LDAP bind credentials or browser-stored tokens.
  CSP, frame denial, nosniff, referrer and permissions headers are applied.
  The DMCA-removed remote web client is not restored.
- **Denial of service:** request/header limits, UI pagination capped at 100 rows and login
  rate controls bound common paths. Deployments must add reverse-proxy rate
  limits for the public audit endpoints and capacity limits for database/log
  storage.
- **Elevation of privilege:** UI authentication checks `IsAdmin` after token
  lookup and account-enabled validation. HTMX mutations for users, groups,
  OAuth providers, and token revocation repeat that check, require CSRF, and
  require explicit confirmation for destructive actions. Other mutations remain
  available only through the inherited authorized administrator API.

## Accepted limitations and external gates

- File-backed admin sessions are suitable for a single instance. Multiple API
  replicas require a shared Goravel session driver before load balancing.
- HTTPS termination, HSTS effectiveness, trusted proxy configuration, LDAP/OIDC
  certificates and public-route rate limiting are deployment responsibilities.
- Logged-in RustDesk remote connections require compatible secure-TCP/API-token
  support in hbbs. Official rustdesk-server 1.1.16 does not provide it.
- `switch-grant` remains absent until device public-key discovery/storage,
  signature verification and replay protection are all available.
- No test creates records on a shared development or production database; all
  database tests use temporary or in-memory SQLite and clean up automatically.

Framework references:

- [Goravel session documentation](https://www.goravel.dev/the-basics/session.html)
- [Goravel v1.18 CSRF middleware](https://github.com/goravel/framework/blob/v1.18.0/http/middleware/verify_csrf_token.go)
- [HTMX documentation](https://htmx.org/docs/)
