# RustDesk compatibility

Verified on 2026-09-05 against RustDesk client `1.4.9`
(`6c578292e8ebbbec708b76986ba8c4bc7c509747`) and RustDesk server `1.1.16`
(`73523b31cfd25d77dee862e6fc9f5e1fb5e485ef`). These were the latest official
releases at verification time.

For the broader Server Pro administration surface, see the
[official API parity audit](official-api-parity.md). Client route coverage alone
is not full official API conformance.

## Contract status

| Surface | Status | Evidence |
|---|---|---|
| Login, login options, OIDC callbacks, heartbeat and sysinfo | Preserved through the inherited Gin router behind Goravel's Gin driver | `bootstrap/goravel_test.go`; `http/router/api.go` |
| Bearer-token user, peer, group and address-book APIs | Preserved; expired tokens and disabled users are rejected | `http/middleware/rustauth_compat_test.go` |
| Usernames | Administrator create/register and RustDesk login accept up to 64 characters | `http/request/admin/user.go`; `http/request/api/login.go` |
| Alias filters | Peer and address-book administrator/personal lists use parameterized alias filters | `peer_compat_test.go`; `addressbook_compat_test.go` |
| Connection and file audit | Existing unauthenticated RustDesk audit routes are preserved; session IDs retain full uint64 precision | `POST /api/audit/conn`; `POST /api/audit/file` |
| Alarm audit | Added for 1.4.9 with strict schema and type validation; visible read-only at `/_admin/alarms` | `audit_alarm_test.go`; `POST /api/audit/alarm` |
| Released client route manifest | Account, group, address-book, heartbeat, sysinfo, OIDC and audit method/path pairs are locked by a focused regression | `TestRustDesk149SupportedClientRoutesAreRegistered` |
| Connection audit notes | RustDesk Server Pro boundary | RustDesk 1.4.9 publicly defines the `GET /api/audit/conn/active` and `PUT /api/audit` request/response shapes, but not the Server Pro GUID lifecycle, authorization, or persistence semantics; this fork does not invent them. |
| Explicit device deployment | RustDesk Server Pro boundary | `POST /api/devices/deploy` requires a Pro API token with Devices read/write permission; this fork does not emulate that authorization model. |
| Command-line device assignment | RustDesk Server Pro boundary | RustDesk 1.4.9 `--assign` posts bearer-authenticated device metadata to `POST /api/devices/cli`; this fork does not emulate Pro token or assignment semantics. |
| Recording upload | Dormant client code, not a released default contract | RustDesk 1.4.9 contains `POST /api/record`, but its private `ENABLE` flag defaults false and has no setter in the released source. |
| `switch-grant` | Deliberately not implemented | Current client requests are device-signed, but this API has no durable device public key or replay state with which to verify them. An accept-all route would weaken authentication. |
| Logged-in secure TCP | External server capability gate | The client requires secure TCP when both key and access token exist; official server 1.1.16 lacks the matching API-token flow. |

The route manifest is sourced directly from the released client implementations:

- [Account routes](https://github.com/rustdesk/rustdesk/blob/1.4.9/flutter/lib/models/user_model.dart)
- [Users, peers and device groups](https://github.com/rustdesk/rustdesk/blob/1.4.9/flutter/lib/models/group_model.dart)
- [Legacy and structured address books](https://github.com/rustdesk/rustdesk/blob/1.4.9/flutter/lib/models/ab_model.dart)
- [Heartbeat and sysinfo](https://github.com/rustdesk/rustdesk/blob/1.4.9/src/hbbs_http/sync.rs)
- [OIDC](https://github.com/rustdesk/rustdesk/blob/1.4.9/src/hbbs_http/account.rs)
- [Connection, file and alarm audits](https://github.com/rustdesk/rustdesk/blob/1.4.9/src/server/connection.rs)

Route registration proves the HTTP contract is present; handler behavior remains
covered by the focused controller, middleware, request and service tests plus the
full Go suite.

## Alarm contract

RustDesk builds the endpoint as `{api-server}/api/audit/alarm` and posts JSON
with `Content-Type: application/json` and no authorization header. The stable
payload is:

```json
{
  "id": "controlled device id",
  "uuid": "base64 device uuid",
  "typ": 0,
  "info": "JSON encoded as a string",
  "conn_id": 123,
  "conn_audit_ref": "optional; IP-whitelist alarms only"
}
```

Accepted `typ` values are `0`, `1`, `2`, `6`, `7`, `8`, and `9`. The API
rejects other values, oversized identifiers/information, malformed inner JSON,
and database failures. Because the official client sends no credential, alarm
records are security telemetry rather than authenticated proof of device
identity; rate-limit this public write route at the reverse proxy.

Official source:

- [Audit URL construction](https://github.com/rustdesk/rustdesk/blob/1.4.9/src/common.rs#L1118-L1124)
- [Alarm payload and POST](https://github.com/rustdesk/rustdesk/blob/1.4.9/src/server/connection.rs#L1494-L1516)
- [Alarm type values](https://github.com/rustdesk/rustdesk/blob/1.4.9/src/server/connection.rs#L6048-L6060)

Additional official boundaries:

- [Connection audit GUID lookup](https://github.com/rustdesk/rustdesk/blob/1.4.9/flutter/lib/models/model.dart#L1242-L1318)
- [Connection audit note update](https://github.com/rustdesk/rustdesk/blob/1.4.9/flutter/lib/common/widgets/dialog.dart#L1656-L1686)
- [RustDesk Server Pro explicit deployment](https://rustdesk.com/docs/en/self-host/client-deployment/#explicit-deployment-for-new-devices)
- [Command-line device assignment](https://github.com/rustdesk/rustdesk/blob/1.4.9/src/core_main.rs#L563-L638)
- [Dormant recording uploader](https://github.com/rustdesk/rustdesk/blob/1.4.9/src/hbbs_http/record_upload.rs#L16-L27)

## Secure-TCP boundary

RustDesk 1.4.9 calls `secure_tcp` whenever both the server key and login token
are present ([client source](https://github.com/rustdesk/rustdesk/blob/1.4.9/src/client.rs#L429-L433)).
The proposed open-source server support in
[rustdesk-server PR #400](https://github.com/rustdesk/rustdesk-server/pull/400)
was closed without merge. This API therefore preserves key and signature
verification and does not claim that local HTTP tests prove a logged-in remote
session against the official server.

## Verification levels

- Local: Go tests, build, vet, vulnerability scan, route fallback, session,
  CSRF, static asset and alarm persistence checks.
- Browser (verified 2026-08-10): login/logout, captcha trigger, protected
  redirects, HTMX filtering and empty state, management forms, clean console,
  and responsive layouts at 320/768/1024/1440 pixels.
- External gate: a physical RustDesk 1.4.9 client, a server with the required
  secure-TCP/token capability, production TLS/reverse proxy and persistent
  storage. These are not inferred from local success.

## Live transport validation — 2026-09-10

The maintained [server fork](https://github.com/slxar/rustdesk-server-secure-tcp/commit/15e65e4)
now retains TCP/WebSocket registrations, sends the client-required heartbeat,
answers WebSocket online queries and routes inbound messages over registered
streams. Disposable wire tests covered encrypted TCP nonce continuity, signed
peer keys, reconnect ownership and registration beyond 30 seconds. An official
RustDesk 1.4.9 Linux client subsequently connected inbound to a 1.4.9 Mac through
the public TLS WebSocket relay and received desktop video. Existing server keys
and peer records were preserved and temporary test data was cleaned up.

That connection evidence supersedes the earlier locked-desktop gate for this
scenario. It does not establish every Pro API or policy behavior.
