# RustDesk API Goravel/HTMX Checklist

## Discovery and contracts

- [ ] Audit all 433 upstream issues and classify all 106 open issues
- [ ] Record RustDesk 1.4.9 / server 1.1.16 API compatibility matrix
- [ ] Record current-master additive deltas
- [ ] Record security trust boundaries and external secure-TCP gate

## Reproducible baseline

- [ ] Add generator build exclusions
- [ ] Commit and review `go.sum`
- [ ] Remove private-LAN Redis test dependency
- [ ] Pass inherited `go test ./...`, `go build ./...`, and `go vet ./...`

## Goravel foundation

- [ ] Pin Goravel/Gin driver 1.18.0 and Go 1.25
- [ ] Add application bootstrap, config, facades, and graceful shutdown
- [ ] Add the one-way Gin compatibility adapter
- [ ] Register all RustDesk API routes
- [ ] Register all inherited admin API routes
- [ ] Add route and response contract tests

## HTMX admin

- [ ] Self-host HTMX 2.0.10 with license/integrity provenance
- [ ] Add secure login/logout, session rotation, and CSRF
- [ ] Add dashboard and responsive navigation
- [ ] Add user, peer/alias, group, address-book/tag workflows
- [ ] Add audit/token/OAuth/LDAP/config workflows
- [ ] Add keyboard, empty, loading, and error states

## Actionable issues

- [ ] Build/test reproducibility (#529)
- [ ] Data durability/backups (#336, #424, #491, #532)
- [ ] Alias filter (#370, #400, #527)
- [ ] Username length (#466)
- [ ] LDAP diagnostics/group admission (#372, #508, #509)
- [ ] Disabled-user/token expiry behavior (#133, #429, #479)
- [ ] Optional TOTP (#419, #472, #489)
- [ ] Static/admin security and web-client state leak (#427, #500)
- [ ] Audit immutability/retention (#474)
- [ ] RustDesk 1.4.9 audit fields and `/audit/alarm`
- [ ] Additive current-master `/switch-grant` contract
- [ ] Document server-owned group ACL/multi-relay/secure-TCP gates

## Release verification

- [ ] Focused tests pass after each slice
- [ ] Full `go test ./...`
- [ ] `go build ./...`
- [ ] `go vet ./...`
- [ ] Native Go vulnerability/dependency audit is triaged
- [ ] `git diff --check`
- [ ] Browser console/network/keyboard/responsive checks pass
- [ ] Container restart preserves local test data
- [ ] No shared dev/prod test records were created
- [ ] Codebase-memory graph is refreshed
- [ ] Independent xhigh review is approved
- [ ] Cognee durable memory is synchronized, or its 404 blocker is reported

## Publication

- [ ] Create the new public GitHub repository
- [ ] Preserve MIT license and upstream attribution
- [ ] Push verified default branch and tag
- [ ] Verify remote SHA, visibility, default branch, and release URL
