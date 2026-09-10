# RustDesk browser session

This is an AGPLv3 browser session transport derived from
[AllenMGu/webclient-v1](https://github.com/AllenMGu/webclient-v1/tree/6a77fe146336ffefe9636af27422bd472c6a192e)
and its RustDesk V1 TypeScript bridge. It uses a native HTML/canvas UI and VP9
WASM decoding; it is not the proprietary Server Pro WebClient V2.

Build with Node.js >=22.12:

```sh
npm ci
npm test
npm run build
```

The build writes all runtime files and the complete modified source archive to
`../resources/webclient`. Serve this asset directory through the protected API
webclient handler; it must not be exposed as a general static directory.
The source archive can be made publicly accessible independently of live sessions.

`GET /webclient/session-config` supplies `{id,key,id_ws,relay_ws}` for the existing
HTTP-only cookie session. Relative WebSocket paths resolve to same-origin ws/wss.
Only `/webclient/ws/` paths are accepted. The gateway must independently enforce
the target and bind each relay UUID to a signed rendezvous response for that target.
The browser does not receive a reusable account token or accept editable server/ID
parameters. Disconnect posts JSON to `/webclient/logout` and closes the transport.

The host still authenticates its password or local approval. The password is
challenge-hashed in memory and never stored. Server-signed peer identity and
peer-signed encryption keys must verify; missing/invalid signatures abort with
no downgrade. Video, mouse, keyboard, wheel, display switching and full screen
are supported along with the toolbar, clipboard and file-transfer controls below.
Audio, directory transfers and terminal sessions are not implemented.

CSP requires same-origin scripts, styles, connections and workers, plus
`script-src 'wasm-unsafe-eval'` and `connect-src data:` for the bundled Zstd WASM
initializer. No external CDN or font requests are needed.

The decoder assets come from the original RustDesk V1 source archive in the
upstream project. OGV.js and libvpx notices are preserved under `static/ogvjs-1.8.6`.
JavaScript dependency licenses accompany the source archive and runtime output.
`tests/security.ts` verifies the complete signed encryption handshake, wrong-key
and wrong-target rejection, first-message protocol IDs, and endpoint binding.

The unused OGVPlayer style injection is removed from ogv.js so decoder loading
works under the strict stylesheet CSP without permitting inline styles.

The session toolbar includes display/quality/view options, authenticated host
controls and an explicit clipboard dialog. File transfer uses a second encrypted
connection restricted to the same host; the host must permit and authenticate it.
The native-style file manager has local/remote panes, navigation, filtering,
sorting, multi-selection and a serial transfer queue. Local files/folders must be
selected using the browser picker. Regular files up to 64 MiB can be uploaded/downloaded; existing files are
refused and directories are browsable but not transferred. See
[`docs/webclient.md`](../docs/webclient.md) for limits and verification boundaries.
