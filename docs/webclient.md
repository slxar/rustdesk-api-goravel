# Authenticated browser connections

Open `/_admin/webclient` after signing in to the API's existing web administration.
Enter one RustDesk ID to create a link. Its recipient can open that device without
portal login, but must still authenticate to the remote host using its password or
local approval. Opening a link shows an explicit **Use access link** button so link
previews do not consume it.

The hosted client supports encrypted VP9 video, keyboard, mouse, scrolling,
display selection, full screen with the toolbar retained, quality/view settings,
Ctrl+Alt+Del (when the host supports it), lock screen, input blocking, view-only
mode, cursor and lock-after-session controls, explicit text clipboard exchange,
and a separate authenticated file manager with local/remote panes and a transfer queue.
The file manager provides back/up/home/refresh, name filtering, sortable name/modified/size
columns, hidden remote files, multi-file selection and serial Send/Receive jobs. Local
access requires selecting files or granting a browser folder picker; received files
are offered as browser save links. See the [native interface audit](native-file-transfer-parity.md)
for the complete control inventory and remaining differences. It is an AGPLv3 client derived from the
RustDesk V1 transport, not the proprietary Server Pro WebClient V2. Audio, clipboard file copy, directory transfers and destructive file-management
actions are not implemented. Single-file transfers are limited to 64 MiB; existing
destinations are refused rather than overwritten. The corresponding modified source and
third-party notices are always available at `/webclient/source.tar.gz`.

## API

See the [integration API guide](integration-api.md) for the complete request contract,
error responses and backend examples.

For another system, open `/_admin/integration-tokens` and create a named
integration token. Specify allowed IDs separated by commas/spaces, or explicitly
enter `*` for every target your administrator account can share. Expiry defaults
to 90 days and may be 1–365 days. Copy the `rdapi_...` secret immediately: it is
shown once and only its SHA-256 hash is stored. The interface shows the prefix,
scope, expiry, last-used time and a Revoke action. These tokens authorize only
the access-link endpoint, never general account/admin APIs. Revoking or expiring
a token also invalidates its derived links and browser sessions.

Use that integration token (an existing RustDesk account Bearer token is also
accepted) in the API request:

```sh
curl --fail-with-body https://YOUR_API_HOST/api/webclient/access-tokens \
  -H "Authorization: Bearer $RUSTDESK_ACCESS_TOKEN" \
  -H 'Content-Type: application/json' \
  --data '{"id":"123456789","expires_in":300}'
```

Success is HTTP 201:

```json
{
  "url": "https://YOUR_API_HOST/webclient/open#SINGLE_USE_SECRET",
  "id": "123456789",
  "expires_in": 300,
  "single_use": true,
  "session_expires_in": 3600
}
```

`expires_in` defaults to 300 seconds and must be 1–300. The issuer must be enabled
and either an administrator, the device owner, or have full-control permission
for that address-book entry. Viewing a shared address book alone does not grant
permission to issue links. A host password is never part of this API.

The fragment secret is removed from browser history before redemption. It is not
sent in an HTTP URL, access log, referrer, or persistent browser storage. The
browser POSTs it once to `/webclient/redeem`, which atomically consumes its SHA-256
hash and creates a Secure, HttpOnly, SameSite=Strict cookie restricted to
`/webclient`. Replays fail with HTTP 403. A session lasts up to one hour. Explicit
Disconnect revokes it; disabling the issuer or removing its target permission
also stops it. Active sockets check revocation every five seconds.

Integration tokens persist in the additive `integration_tokens` table (database
version 267). Existing users, devices, host credentials and audit rows are retained.
Browser capabilities are intentionally ephemeral and held by one API process. Restarting
the API revokes pending links and browser sessions, while existing user/device
databases are unchanged. Deploy one API replica; a shared atomic store is required
before scaling to several replicas. The store caps pending links and sessions at
10,000 and each session at four simultaneous sockets.

## Connection boundary

The API serves the client and its configuration only to a redeemed capability.
Unauthenticated visitors to `/webclient/` are sent to the existing admin login.
The public redemption page and source archive contain no device/session data.
Both WebSocket routes enforce the configured HTTPS Origin and cookie:

- `/webclient/ws/id`: permits one strictly parsed PunchHoleRequest for the
  capability's ID and only desktop (0) or file-transfer (1) connection types. It verifies the server-signed peer identity in the response
  and records its relay UUID for one minute.
- `/webclient/ws/relay`: requires that exact ID and atomically consumes a UUID
  observed in that session's signed rendezvous response. The resulting socket
  forwards the encrypted desktop stream, without obtaining the host password or
  session encryption keys.

The gateway reassembles complete WebSocket messages in both directions before
forwarding ciphertext. Forwarding individual continuation frames splits large
encrypted file blocks and causes host decryption errors. The regression test sends
a 131,103-byte fragmented message with interleaved ping frames.

The file browser opens its own signed encrypted connection and authenticates to
that same host using `LoginRequest.file_transfer`. Host file-transfer permission
is still required. It does not reuse desktop login as permission to read files.
Protocol fields are checked against RustDesk 1.4.9's pinned hbb_common revision
`7e1c392c62d39c364127307cd408421dd5f8cfb0`: login version field 11 enables overwrite
confirmation, and receive total-size field 5 reports upload size. Uploads wait for
host confirmation before sending contents; downloads validate size and completion.
Cancelling an upload may leave a partial file on the host; remove or rename it
before retrying. Clipboard text is received into the dialog; writing the local
clipboard requires an explicit click.

The client separately verifies server-signed peer identity and peer-signed
encryption keys. Any missing/invalid signature fails closed; there is no plaintext
downgrade. Changing URL parameters, JavaScript, the protobuf target or the relay
UUID does not change a capability's authorization.

Existing public `/ws/id`, `/ws/relay` and native RustDesk ports remain the native
client service. This feature scopes browser capabilities; it does not turn the
public rendezvous service into a private network or replace host authentication.

## Build and reverse proxy

```sh
npm --prefix webclient ci
npm --prefix webclient test
npm --prefix webclient run build
go test -race ./service ./http/controller/web ./http/router ./bootstrap
docker build -f Dockerfile.source -t rustdesk-api-goravel:webclient .
```

Set the existing `rustdesk.api-server` to the public HTTPS origin and
`rustdesk.key` to the server public key. The gateway uses Docker service names
`ws://hbbs:21118` and `ws://hbbr:21119` by default. Other layouts can set
`RUSTDESK_WEBCLIENT_ID_WS` and `RUSTDESK_WEBCLIENT_RELAY_WS` to administrator-chosen
internal WebSocket URLs; browser input cannot choose an upstream address.

Add this Plesk nginx location. Replace the documentation-only backend address
`192.0.2.10` with your API backend IP:

```nginx
location ^~ /webclient/ {
    proxy_pass http://192.0.2.10:21114;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "Upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 3600s;
    proxy_send_timeout 3600s;
    proxy_buffering off;
}
```

The `^~` location also prevents Plesk static-extension rules from intercepting
protected JavaScript, CSS and WASM requests. Keep nginx caching disabled here.
Browser traffic uses existing HTTPS port 443; no new public port is needed.
Preserve the API data volume and RustDesk key/database volume during replacement.
Set `SESSION_FILES=/app/data/admin-sessions` after copying existing administrator
session files there to retain admin logins across future image replacements.

## Verification

`service/integration_token_test.go` checks hash-only storage, token scope,
issuer status, expiry and revocation. `http/controller/api/web_access_test.go`
verifies HTTP minting, target restrictions, denial at other API routes and
revocation of derived browser access. `service/web_access_test.go` exercises concurrent single-use redemption,
expiration, revocation and relay replay. `http/controller/web/webclient_test.go`
exercises the real WebSocket proxy against isolated upstreams, origin/cookie
checks, invalid signatures, alternate IDs, duplicate/malformed protobuf fields,
unrelated relay UUIDs and disabled issuers. `webclient/tests/security.ts` checks
the signed encrypted handshake and its failure paths. The bootstrap check runs
a WebSocket beyond the normal HTTP timeout: these two socket routes explicitly
omit Goravel's buffered three-second HTTP timeout and use capability deadlines.
The legacy adapter preserves response-writer middleware and starts a fresh Gin
request, so raw/static successful responses do not inherit the fallback's 404.
Tests use disposable data.

### Live deployment check (2026-09-10)

The integration-token build was deployed behind Plesk with persistent API data
and administrator sessions. Live checks covered token creation, target-scoped
link issuance, rejection at unrelated API routes, redemption without portal
login, and token revocation through the management page. The revoked browser
capability returned to the authenticated launcher. The temporary token and the
two connection-audit rows from the browser authentication checks were deleted.
Both SQLite databases passed integrity checks; pre-deployment users, peers and
account tokens remained present, and both server key files matched the backup.

The real RustDesk 1.4.9 host completed rendezvous, relay and the signed encrypted
handshake and presented the browser password prompt. Real-host browser video was subsequently verified after local approval. Real-host
keyboard/control actions remain untested to avoid disturbing the active desktop. The
isolated encrypted-host fixture exercised VP9 rendering, password retry and
keyboard/mouse transport; this does not replace real-host acceptance testing.

### Toolbar and file-transfer verification (2026-09-10)

The browser checks used an isolated signed/encrypted host fixture: desktop video,
Ctrl+Alt+Del, input block/unblock acknowledgments, view-only control gating, image
quality, clipboard exchange, directory browsing, download and a byte-for-byte
262,151-byte upload passed. Desktop and 390px mobile layouts were checked. The
file-input upload used the Codex in-app browser because the external browser's
file-picker automation rejected file selection. Protocol tests cover empty files,
zstd, denial, size bounds, incomplete streams, cancellation, timeout and upload
confirmation. Real-device file-transfer acceptance is recorded below. This is a compatible AGPL browser client,
not full proprietary Server Pro WebClient V2 feature parity.

Legacy Flutter service workers can serve retired client assets from browser cache.
The public access-link landing script unregisters service workers scoped to
`/webclient/` before redemption/navigation. The destination has a fresh cache key
and vendor script URLs are versioned to avoid the old HTTP cache too. It leaves cookies, local storage,
unrelated service workers and server data untouched.
