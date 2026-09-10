# Changelog

## [3.0.1] - 2026-08-10

### Added

- Executable route-contract coverage for the supported RustDesk 1.4.9 client API surface.

### Changed

- Compatibility documentation now distinguishes supported client routes from RustDesk Server Pro device assignment/deployment, connection-audit-note, dormant recording-upload, and unreleased `switch-grant` contracts.

## [3.0.0] - 2026-08-10

### Added

- Goravel 1.18 HTTP lifecycle with the inherited RustDesk API behind the Gin adapter.
- Server-rendered HTMX administrator pages with protected sessions, CSRF, filtering, pagination, selected management forms, and self-hosted HTMX 2.0.10.
- RustDesk 1.4.9 alarm-audit ingestion and administrator visibility.
- Consistent SQLite backup command and restore guidance.

### Changed

- Go 1.25 is now required.
- SQLite uses WAL, full synchronization, foreign keys, and a five-second busy timeout.
- The module path is `github.com/slxar/rustdesk-api-goravel/v3`.

### Fixed

- Alias filters cover peer and address-book administrator/personal lists.
- Usernames up to 64 characters are accepted at the shared RustDesk and administrator boundaries.
- Expired tokens and disabled accounts are rejected on protected API requests.

### Removed

- Legacy `/webclient`, `/webclient2`, bundled web-client assets, web-client configuration, and web-client-only API routes.
- The external administrator SPA build dependency.

### Security

- Removed the public legacy `/_admin` static tree.
- Administrator cookies are Secure, HttpOnly, SameSite=Lax, and scoped to `/_admin`.
- RustDesk key/signature/E2EE verification is never bypassed; compatible server support remains required for logged-in secure TCP.
