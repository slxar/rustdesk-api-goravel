# RustDesk 1.4.9 file-transfer interface audit

Source audit: 2026-09-10. This compares the official **native file manager and remote-session toolbar** with this repository's browser implementation. It is not a claim that every native RustDesk feature is available in a browser.

## Reference implementation

- [Desktop file-manager page](https://github.com/rustdesk/rustdesk/blob/1.4.9/flutter/lib/desktop/pages/file_manager_page.dart): layout, navigation, selection, file operations and transfer-status cards.
- [File model](https://github.com/rustdesk/rustdesk/blob/1.4.9/flutter/lib/models/file_model.dart): directory history, transfer jobs, conflict handling, cancellation and saved-job resumption.
- [Host connection](https://github.com/rustdesk/rustdesk/blob/1.4.9/src/server/connection.rs): permission checks and file-action processing.
- [Pinned wire schema](https://github.com/rustdesk/hbb_common/blob/7e1c392c62d39c364127307cd408421dd5f8cfb0/protos/message.proto) and [transfer engine](https://github.com/rustdesk/hbb_common/blob/7e1c392c62d39c364127307cd408421dd5f8cfb0/src/fs.rs): RustDesk 1.4.9's `hbb_common` submodule.

## Controls and behavior

| Official native feature | Browser adapter status / practical requirement |
| --- | --- |
| Local pane, remote pane, transfer-status sidebar | Implemented. The native `build` method uses three columns; its own `isWeb` branch omits the unrestricted local pane. Browser local access comes from an explicit file/directory picker. |
| Back, parent directory, Home, refresh, typed location, breadcrumbs | Back/up/home/refresh and remote typed-path navigation are implemented. Local navigation is limited to selected files or granted directory handles. Native breadcrumb menus are not reproduced. |
| Search current listing | Implemented locally in both panes; no network protocol change. |
| Name, modified time, size; ascending/descending sorting | Implemented using `FileEntry` fields and local UI sorting. |
| Resizable columns | Local UI behavior; no server changes. |
| Hidden files | Remote toggle implemented through `read_dir.include_hidden`. Browser local pickers may filter files according to browser/OS policy. |
| Multi-selection, select/unselect all, range selection | Individual checkbox multi-selection is implemented; selected files enter a sequential queue. Native select-all/range-selection behavior is not fully reproduced. |
| Send/receive selected files | Individual upload and download are implemented with host authentication and permissions. Browser downloads produce a save link; uploads read the selected `File`. |
| Upload/download directories | Not implemented by the adapter. Requires recursive enumeration, safe relative-path validation and multi-file job handling. Folder picking alone does not implement directory transfer. |
| Drag-and-drop | Can supply explicitly dropped local files. It cannot grant unrestricted local filesystem access. |
| Create folder | Native sends `FileAction.create`. Not implemented by the browser adapter. Requires name validation, response correlation and refresh. |
| Delete files/directories | Native sends `remove_file`, `remove_dir` and recursive enumeration, with confirmation. Not implemented by the browser adapter. |
| Rename | Native context menu enables it for peers at least 1.3.0 and sends `FileAction.rename`. Not implemented; the older browser schema also needs the corresponding field. |
| Existing-file skip/overwrite and apply-to-batch choice | Native uses digest comparisons and explicit conflict decisions. Browser refuses existing destinations, including identical-file skip responses; rename the local file to upload a new destination. |
| Transfer direction, file name, bytes/progress, status/errors | Adapter emits direction, byte counts, completion and error events for the UI. |
| Multiple jobs and retained status cards | Implemented as a sequential queue and retained status cards with a clear-completed control. The adapter still runs one active job; this does not imply concurrent transfers. |
| Cancel/remove job | Implemented with `FileAction.cancel`; an in-flight read or backpressure wait cannot restart the cancelled job. |
| Resume paused/saved jobs | Native stores job metadata and supports `offset_blk`. Browser refuses nonzero resume offsets and does not persist host passwords or transfer sessions. |
| Very large files | Current browser adapter limits each file to 64 MiB. Downloads buffer data and construct a Blob; browser memory overhead can exceed the payload size. Raising this limit requires a streaming destination. |

## Protocol details that must remain correct

File management is a **separate authenticated connection** to the same authorized RustDesk ID. Setting `enable_file_transfer` on a remote-desktop connection does not enable the standalone file manager; that option controls file clipboard behavior. The rendezvous request uses `ConnType.FILE_TRANSFER = 1`, and `LoginRequest.file_transfer` supplies the initial folder and hidden-file choice. The login omits desktop options and includes `version = "1.4.9"` in protobuf field 11 so the host enables overwrite detection. The API gateway allows only desktop/file-transfer connection types and preserves target/key/relay authorization checks.

Uploads send `FileAction.receive`, including `total_size` in field 5, followed by a digest. File bytes are withheld until the host sends `send_confirm.offset_blk = 0`. An existing-file digest or skip/resume response is rejected. Blocks are at most 128 KiB and uncompressed; even an empty file sends an empty block so the host creates it. Success requires the host's final `FileResponse.done` acknowledgment.

Downloads send `FileAction.send`. The adapter requires exactly one regular-file entry, validates its size and name, confirms the digest, and accepts ordered file-zero blocks. RustDesk may compress blocks with Zstandard; decoding is bounded to a 128 KiB output block. The adapter checks total received bytes before accepting completion. Official sender blocks currently leave `blk_id` at zero; it is not a byte offset. A one-file job finishes with `done.file_num = 1` after advancing beyond file zero.

Host `PermissionInfo.File` and host account/OS permissions still apply. The browser's API capability bypasses portal login only. API token revocation, capability expiry and the encrypted relay binding continue to apply to both connections.

The API gateway must forward complete WebSocket **messages**, not individual frames. Browser uploads can fragment a 131,103-byte encrypted block into 64 KiB continuation frames. The previous `golang.org/x/net/websocket.Message.Receive` path forwarded the first 65,536 bytes as a separate encrypted message, causing native host decryption failure. The gateway now uses Gorilla's `ReadMessage`/`WriteMessage` to reassemble both directions, with the existing authentication checks and aggregate message limits retained. Reducing upload chunks would only conceal this transport error.

## Remote-session toolbar audit

Reference: [desktop toolbar](https://github.com/rustdesk/rustdesk/blob/1.4.9/flutter/lib/desktop/widgets/remote_toolbar.dart) and [shared toolbar actions](https://github.com/rustdesk/rustdesk/blob/1.4.9/flutter/lib/common/widgets/toolbar.dart). Native menus are conditional on platform, negotiated features, host permission, session type and sometimes peer version. A visible label alone does not establish working parity.

| Official native control group | Browser implementation / gap |
| --- | --- |
| Pin, collapse, draggable docking and multi-edge toolbar | A persistent browser toolbar is present; native docking/pinning state is not reproduced. |
| Fullscreen and view scale | Browser fullscreen, fit and original-size view are implemented. Native custom-scale slider, OS-window resizing and edge-scroll modes are not implemented. |
| Display selection/cycling, all displays, individual display windows | Single-display selection is implemented. Simultaneous all-display composition and native multi-window display management are not. |
| Quality presets and custom quality/FPS | Low/balanced/best presets are implemented. Custom quality/FPS and detailed quality-monitor UI are not. |
| Codec choice and true color 4:4:4 | Browser transport currently decodes VP9. VP8/AV1/H264/H265 selection and 4:4:4 negotiation are not implemented. |
| Remote resolution and virtual monitors | Not implemented; requires negotiated display capabilities and corresponding request/response handling. |
| Remote cursor; follow cursor/window; cursor scaling | Show-remote-cursor option is implemented. Follow-window/cursor and native cursor-zoom controls are not. |
| Keyboard/mouse, view-only mode, Ctrl+Alt+Del and lock | Implemented with host login/keyboard permission checks and browser-local view-only suppression. Browser-reserved shortcuts cannot always be captured; native raw keyboard modes/layout/input-source selection are not reproduced. |
| Clipboard text and send clipboard as keystrokes | Manual local clipboard read, remote clipboard display/copy, send clipboard text and type-text actions are implemented. Native file clipboard/drag-copy protocols are separate and not implemented. |
| Reverse wheel, swap mouse buttons/control-command keys, relative mouse, trackpad speed and mobile actions | Not implemented as native-equivalent settings. |
| Block/unblock host input | Protocol option and host acknowledgment are implemented. The UI and connection enforce Windows, keyboard permission and `PermissionInfo.BlockInput = 7`; view-only also disables the UI action. |
| Lock after disconnect | Protocol option implemented with keyboard permission, view-only UI gating and the native Android exclusion. |
| Request elevation, OS account/password, restart and privacy mode | Not implemented. These need distinct negotiated capabilities, permission checks and explicit action flows; they are not substitutes for ordinary text input. |
| Transfer files | Separate same-ID file-transfer connection implemented; see the file-manager table. |
| View camera, terminal and TCP tunneling | Not implemented. They use separate connection scopes; the API capability gateway currently rejects these scopes. |
| Switch sides | Not implemented; the browser is not a native RustDesk host accepting incoming control. |
| Refresh video | Implemented. |
| Chat and voice call/audio output | Not implemented. The transport has some inherited message definitions, but there is no complete audio/chat UI and media pipeline. Official native toolbar itself hides voice/recording in its `isWeb` branch. |
| Session recording and screenshots | Native-equivalent recording and remote screenshot workflow are not implemented. Browser canvas capture would be a distinct feature with explicit user controls. |
| Account notes and connection-audit notes | Not implemented by the browser session UI; API account/Pro parity must be assessed separately. |

The API remains an authorization gateway for encrypted client-to-host traffic. Adding toolbar menus does not require opening additional public ports; implementing a new connection type would require deliberately extending the gateway's scope checks.

## Validation boundary

`cd webclient && npm test` exercises wire round trips, file login and permissions, plain/Zstandard/empty downloads, empty/chunked uploads, destination confirmation, size/name/type bounds, cancellation, host errors, timeouts and consecutive encrypted frames. These are isolated protocol checks. A complete native-parity claim additionally requires implemented UI actions and real-host end-to-end validation for each feature; unsupported rows above must remain visible in release notes.

`go test ./http/controller/web ./bootstrap` additionally checks capability scope/replay, exact burst delivery, fragmented messages in both directions with interleaved ping frames, and WebSocket lifetime beyond the ordinary HTTP timeout. The fragmentation regression failed against the previous gateway (65,536 bytes received instead of 131,103) and passes after migration. Live upload confirmation remains a separate deployment check.
