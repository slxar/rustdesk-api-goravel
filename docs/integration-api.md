# Integration API: single-use remote access links

Use this API from your application's **backend** to create a browser access link
for one RustDesk ID. The recipient does not need to log in to the RustDesk portal.
The remote host's password or local approval is still required. Creating a link
does not check whether the host is online or start a remote desktop session.

Example endpoint (replace `rustdesk.example.com` with your API origin): `POST https://rustdesk.example.com/api/webclient/access-tokens`

## 1. Create an integration API token

Sign in to [Integration API tokens](https://rustdesk.example.com/_admin/integration-tokens).
Create a named token for your system, with a list of permitted RustDesk IDs
separated by commas or spaces. Use `*` only when the integration needs access to
every host its administrator can share. Expiry is 1–365 days, initially 90 days.

Copy the full `rdapi_...` token when it is shown; it cannot be retrieved later.
Store it in your backend's secret manager or environment, never in browser code.
The management page lists token prefixes, allowed IDs, expiry, last use and
revocation. Integration tokens authorize only this access-link endpoint.

For rotation, create a replacement, update your integration, then revoke the old
token. Revocation also invalidates links and browser sessions issued by that token.

## 2. Request a link

Your backend must authenticate its own caller and verify that caller may access
the requested host before calling this endpoint. An integration token represents
your system, not each of its end users.

```sh
# Set RUSTDESK_API_TOKEN securely in your backend environment first.
curl --fail-with-body --request POST \
  'https://rustdesk.example.com/api/webclient/access-tokens' \
  --header "Authorization: Bearer $RUSTDESK_API_TOKEN" \
  --header 'Content-Type: application/json' \
  --data '{"id":"123456789","expires_in":300}'
```

Replace the example ID with your permitted host ID. Header prefix `Bearer ` is
case-sensitive. The JSON body is limited to 1,024 bytes.

| JSON field | Type | Required | Contract |
| --- | --- | --- | --- |
| `id` | string | Yes | One exact RustDesk ID, 6–64 characters from `A-Z`, `a-z`, `0-9`, `_`, `-`. Must be permitted by the token and issuer. Do not send a JSON number or an ID list. |
| `expires_in` | integer | No | Link lifetime in seconds: 1–300. Omitted or `0` means 300. |

Success: **HTTP 201**, with `Cache-Control: no-store`:

```json
{
  "url": "https://rustdesk.example.com/webclient/open#SINGLE_USE_SECRET",
  "id": "123456789",
  "expires_in": 300,
  "single_use": true,
  "session_expires_in": 3600
}
```

| Response field | Meaning |
| --- | --- |
| `url` | Complete access link to give to the authorized recipient. Preserve the `#` fragment. |
| `id` | Host authorized by this link. It cannot be changed by editing the URL. |
| `expires_in` | Link lifetime from issuance, in seconds. |
| `single_use` | Always `true`; only one redemption succeeds. |
| `session_expires_in` | Maximum browser session lifetime, 3,600 seconds from redemption, subject to revocation. |

Treat the returned URL as a secret. Do not log it, put it in analytics, or cache
the response. Return it only to the authorized user and open it as a normal
browser navigation. Do not remove its fragment or try to redeem it from your
backend: redemption establishes the recipient's browser cookie.

### Node.js backend example (built-in fetch)

Call this function only after your application's user/host permission check.

```js
async function createRustDeskLink(rustdeskId) {
  const token = process.env.RUSTDESK_API_TOKEN;
  if (!token) throw new Error('RUSTDESK_API_TOKEN is not configured');

  const response = await fetch(
    'https://rustdesk.example.com/api/webclient/access-tokens',
    {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ id: String(rustdeskId), expires_in: 300 }),
      signal: AbortSignal.timeout(10000),
    },
  );
  if (response.status !== 201) {
    // Log status only; never log credentials or successful response URLs.
    throw new Error(`RustDesk link request failed (${response.status})`);
  }
  return await response.json();
}
```

## Errors and retries

These are the JSON errors returned by this endpoint when using an integration
token. A reverse proxy or network failure may return a different body, so check
the HTTP status before assuming JSON.

| Status | JSON body | Action |
| --- | --- | --- |
| 400 | `{"error":"invalid request"}` | Check JSON syntax, field types, required `id` and the 1,024-byte body limit. |
| 400 | `{"error":"expires_in must be 1 to 300 seconds"}` | Correct the requested lifetime. |
| 401 | `{"error":"Unauthorized"}` | Check the exact Bearer header and token validity. The token may be expired/revoked or its issuer no longer eligible. |
| 403 | `{"error":"target access denied or web client is not configured"}` | Check the ID format, token allowlist, issuer permission and server HTTPS/public-key configuration. Issuance capacity or other issuance failures also use this response. |

Do not automatically retry 400/401/403 responses without correcting the cause.
There is **no idempotency key**: every successful POST creates a new link. If a
request times out after issuance, retrying may leave an unused link that expires
automatically within five minutes. Request links just before use rather than
pre-generating them. HTTP 201 does not guarantee that the host is online or that
its password will be accepted.

## 3. Recipient connection flow

1. Your system gives the complete `url` to the authorized recipient.
2. The recipient opens it and clicks **Use access link**. Loading the landing
   page alone does not consume the link, which avoids consumption by link previews.
3. Redemption consumes the link atomically and establishes a Secure, HttpOnly
   browser session restricted to that host. Reuse or expired redemption fails
   with HTTP 403; request a new link for another attempt.
4. The recipient supplies the host password or obtains host approval. The API
   never returns a host password or bypasses host authentication.
5. The session ends after at most one hour, on explicit disconnect or on
   revocation. Active sockets recheck authorization every five seconds.

The issuer must remain enabled and retain permission to share the host. Expiry
or revocation of the integration token invalidates both pending links and derived
browser sessions. API restarts also invalidate pending links and browser sessions;
the persistent integration tokens remain usable to request new links.

## Deployment and compatibility notes

- External integration and browser traffic use HTTPS port 443; no new public port
  is required for this endpoint. Use the HTTPS origin without the Plesk port.
- The current capability store supports one API replica. Multiple replicas need a
  shared atomic store. Its combined limit is 10,000 pending links/browser sessions.
- The hosted browser client supports VP9 video, keyboard, mouse, scrolling,
  display selection, full screen, control/quality options, explicit text clipboard
  exchange and a native-style file manager with local/remote panes and queued,
  separately authenticated upload/download (one active file up to 64 MiB,
  no overwriting). Audio, clipboard file copy and directory transfers are not
  implemented. Existing native RustDesk host/server configuration is still needed.
- Existing account Bearer tokens are also accepted, but dedicated integration
  tokens are the supported management flow described here. This endpoint is this
  deployment's integration contract, not a claim of Server Pro API parity.

See [web-client operations and authorization details](webclient.md) for proxy
configuration, lifecycle tests and the current real-host verification boundary.
