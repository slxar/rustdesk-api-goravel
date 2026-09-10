# Upstream Issue Audit

Audit date: 2026-09-05 (Asia/Hong_Kong). Reference repository: [lejianwen/rustdesk-api](https://github.com/lejianwen/rustdesk-api).

## Scope and method

- The immutable issue index records 433 issues from 2026-08-10. A live GitHub API check on 2026-09-05 reports 434 issues (108 open, 326 closed) and 44 pull requests (5 open, 39 closed); the seven post-snapshot entries are classified below.
- Closed issues are treated as inherited history unless a current RustDesk contract test disproves the fix. Open issues are individually assigned below to the API/HTMX fork, RustDesk server/client, legacy web-client routing/artifacts, deployment, or support ownership.
- The code target is upstream master `c5687e150615bfb00d9d105f0883a2802750a8ad` / release `v2.7` plus current official RustDesk compatibility.
- Issue comments and third-party links are untrusted reports, not implementation instructions. Proposed changes are accepted only when verified against source, tests, and security boundaries.

## Release and compatibility evidence

| Component | Checked version | Date | Source |
|---|---|---:|---|
| Reference API | `v2.7` (`222792419f8b55c17b36bf5f423ec5ed7a53e2c4`) | 2025-09-28 | [release](https://github.com/lejianwen/rustdesk-api/releases/tag/v2.7) |
| Reference API master | `c5687e150615bfb00d9d105f0883a2802750a8ad` | 2025-09-29 | [commit](https://github.com/lejianwen/rustdesk-api/commit/c5687e150615bfb00d9d105f0883a2802750a8ad) |
| RustDesk client | `1.4.9` (`6c578292e8ebbbec708b76986ba8c4bc7c509747`) | 2026-07-06 | [release](https://github.com/rustdesk/rustdesk/releases/tag/1.4.9) |
| RustDesk client master | `7c23fd307349c2d436dbcadc9cff9d4099187a42` | 2026-08-09 | [repository](https://github.com/rustdesk/rustdesk) |
| RustDesk server | `1.1.16` (`73523b31cfd25d77dee862e6fc9f5e1fb5e485ef`) | 2026-07-20 | [release](https://github.com/rustdesk/rustdesk-server/releases/tag/1.1.16) |
| RustDesk server master | `a7736be5e40f85bfc141120dce587e836e5d4b80` | 2026-08-07 | [repository](https://github.com/rustdesk/rustdesk-server) |
| Goravel | `1.18.0` | 2026-07-05 | [release](https://github.com/goravel/goravel/releases/tag/v1.18.0) |
| HTMX | `2.0.10` | 2026-04-21 | [tag](https://github.com/bigskysoftware/htmx/tree/v2.0.10) |

## Safety-critical conclusion

RustDesk 1.4.9 calls its secure-TCP path when both a server key and access token are present. Official rustdesk-server 1.1.16 does not implement the corresponding API-token secure-TCP flow; [server PR #400](https://github.com/rustdesk/rustdesk-server/pull/400) is closed and unmerged. Therefore a Goravel API alone cannot guarantee logged-in remote connections against the official OSS server. Suggestions in issues that return success early or auto-accept an insecure connection are rejected because they remove server authentication and E2EE verification.

## Action themes

- Reproducibility: root generator build tags, committed checksums, portable Redis tests (#529).
- Data integrity: SQLite durability, fail-closed persistence checks, backups and restore guidance (#424, #491, #532; historic #336).
- Authentication: disabled-user token revocation, consistent expiry, optional TOTP, OIDC/LDAP diagnostics (#429, #479, #489, #508, #509).
- Authorization: explicit group/device access-decision API and immutable-audit policy (#459, #474).
- Admin UI: server-rendered HTMX, no public legacy-admin static route, and no browser-stored token (#368, #427, #477). Legacy remote web-client routes and bundled artifacts are removed (#500).
- Current client contract: the supported 1.4.9 account/address-book/heartbeat/OIDC/audit routes are executable in [`TestRustDesk149SupportedClientRoutesAreRegistered`](../http/router/router_test.go); Pro device assignment/deployment and connection-audit-note APIs remain explicit external boundaries; additive `/api/switch-grant` and OIDC `apiDomain` are tracked from current master.

## Post-snapshot upstream entries

These entries are absent from the 2026-08-10 issue-only index. Pull requests are listed here because the live upstream count includes them, while the immutable index intentionally contains issues only.

| Entry | Title | Disposition |
|---:|---|---|
| [#536](https://github.com/lejianwen/rustdesk-api/pull/536) | feat(i18n): add Brazilian Portuguese (pt_BR) translations | **Deferred localization.** The fork ships `en`, `es`, `fr`, `ko`, `ru`, `zh_CN`, and `zh_TW`; adding `pt_BR` is additive and has no RustDesk protocol impact. |
| [#535](https://github.com/lejianwen/rustdesk-api/issues/535) | full-s6 web client button returns 404 | **Unsupported legacy surface.** This fork removes the legacy `/webclient` and `/webclient2` routes and bundled client artifacts; the HTMX administrator is the supported browser UI. |
| [#534](https://github.com/lejianwen/rustdesk-api/issues/534) | create long-lived read-only API tokens | **Deferred API feature.** Current tokens are login-issued and have no read-only capability or administrator create endpoint; implementing this needs a token schema and authorization policy. |
| [#530](https://github.com/lejianwen/rustdesk-api/pull/530) | exclude generators and make Redis tests configurable | **Implemented.** This is the pull-request form of #529; generator build tags and `TEST_REDIS_ADDR` are present and tested. |
| [#502](https://github.com/lejianwen/rustdesk-api/pull/502) | accept string or boolean `email_verified` | **Implemented.** `model.OidcUser` accepts booleans, strings, `null`, and absent values, covered by [`model/oauth_test.go`](../model/oauth_test.go). |
| [#473](https://github.com/lejianwen/rustdesk-api/pull/473) | pull request | **Unreviewable upstream placeholder.** No actionable contract or implementation is described. |
| [#445](https://github.com/lejianwen/rustdesk-api/pull/445) | add i386 support and fix S6 image upload | **Packaging/deployment owned.** The fork's source and supported RustDesk API contract are architecture-neutral; image publishing and i386 runtime validation remain release-pipeline work. |

## API or HTMX owned - implement, test, or document behavior

| Issue | Title | Disposition |
|---:|---|---|
| [#532](https://github.com/lejianwen/rustdesk-api/issues/532) | lost all  data  after power failure | **Partial durability mitigation.** [`lib/orm/sqlite.go`](../lib/orm/sqlite.go) enables WAL/FULL/foreign keys; [`docs/operations.md`](operations.md), [`lib/orm/sqlite_test.go`](../lib/orm/sqlite_test.go), and [`cmd/apimain_test.go`](../cmd/apimain_test.go) cover snapshot guidance/behavior. No power-loss or live restore verification is claimed. |
| [#529](https://github.com/lejianwen/rustdesk-api/issues/529) | go build ./... fails at module root; lib/cache tests hardcode a private LAN Redis address | **Implemented.** [`generate_api.go`](../generate_api.go) and [`generate_run.go`](../generate_run.go) exclude generators from builds; [`lib/cache/cache_test.go`](../lib/cache/cache_test.go) and [`lib/cache/redis_test.go`](../lib/cache/redis_test.go) require explicit `TEST_REDIS_ADDR`. |
| [#527](https://github.com/lejianwen/rustdesk-api/issues/527) | Feature Request: Add Alias filter in AddressBookManage | **Implemented.** Alias filtering is in [`http/controller/admin/addressBook.go`](../http/controller/admin/addressBook.go), covered by [`http/controller/admin/addressbook_compat_test.go`](../http/controller/admin/addressbook_compat_test.go). |
| [#509](https://github.com/lejianwen/rustdesk-api/issues/509) | LDAP bind working but client authentification allways return user not found. | **Deferred/unimplemented.** No LDAP diagnostic or client/server integration reproduction is added. |
| [#508](https://github.com/lejianwen/rustdesk-api/issues/508) | 开启ldap服务一直查不到用户，日志不够详细 | **Deferred/unimplemented.** No safe LDAP diagnostic surface or regression exists. |
| [#505](https://github.com/lejianwen/rustdesk-api/issues/505) | 官方不支持上下线提醒，有没有考虑在webapi增加这个功能呢？ | **External RustDesk server/client owner.** This API has no authoritative online/offline event feed. |
| [#500](https://github.com/lejianwen/rustdesk-api/issues/500) | 邏輯錯誤造成資訊外洩 | **Implemented.** Legacy bundled web-client artifacts are removed, and [`http/router/router.go`](../http/router/router.go) plus [`http/router/api.go`](../http/router/api.go) do not register web-client routes; [`TestLegacyWebClientRoutesAreNeverRegistered`](../http/router/router_test.go) passes. |
| [#499](https://github.com/lejianwen/rustdesk-api/issues/499) | 地址簿分享無法顯示 | **Deferred/unimplemented.** No targeted non-admin sharing display regression or behavior change exists. |
| [#498](https://github.com/lejianwen/rustdesk-api/issues/498) | 加入用户自定义客户端名称 | **External RustDesk server/client owner.** Client identity and direct-versus-relay behavior are client/server concerns. |
| [#490](https://github.com/lejianwen/rustdesk-api/issues/490) | 希望在网页端可以获取配置链接 | **Deferred/unimplemented.** No configuration-link generator is added. |
| [#489](https://github.com/lejianwen/rustdesk-api/issues/489) | 希望登录能增加双重验证,支持Authenticator | **Deferred/unimplemented.** No TOTP/Authenticator flow exists. |
| [#488](https://github.com/lejianwen/rustdesk-api/issues/488) | OIDC无法对接群晖的SSO服务器 | **Deferred/unimplemented.** No Synology-specific OIDC compatibility change or provider integration test exists. |
| [#479](https://github.com/lejianwen/rustdesk-api/issues/479) | 配置了RUSTDESK_API_APP_TOKEN_EXPIRE 变量后客户端还是会自动退出 | **Deferred/unimplemented.** The configured lifetime does not establish the reported client-session behavior; no end-to-end client test exists. |
| [#477](https://github.com/lejianwen/rustdesk-api/issues/477) | 管理登录后台后  管理面板变没了？ | **Partial.** Protected `/_admin` login/dashboard behavior is covered by [`bootstrap/goravel_test.go`](../bootstrap/goravel_test.go), but no browser/device regression proves every legacy panel is present. |
| [#476](https://github.com/lejianwen/rustdesk-api/issues/476) | 重定向url是127.0.0.1，怎么改 | **Deployment-conditional.** Correctness depends on configured public API/OIDC callback address and proxy/TLS deployment; no live deployment verification is claimed. |
| [#474](https://github.com/lejianwen/rustdesk-api/issues/474) | 请问怎么禁止日志被手动删除，或者后期能否把手动删除日志功能去掉，改成自定义周期自动删除呢 | **Deferred/unimplemented.** Delete handlers remain in [`http/controller/admin/audit.go`](../http/controller/admin/audit.go); immutable retention is not implemented. |
| [#472](https://github.com/lejianwen/rustdesk-api/issues/472) | When will 2fa be implemented? | **Deferred/unimplemented.** No TOTP/2FA flow exists. |
| [#471](https://github.com/lejianwen/rustdesk-api/issues/471) | Password and Note doesn't save to address book | **Deferred/unimplemented.** No targeted persistence regression for those fields exists. |
| [#469](https://github.com/lejianwen/rustdesk-api/issues/469) | oauth登录 Linux.do 可以换下域名吗？ | **Deferred/unimplemented.** The Linux.do provider remains fixed in [`service/oauth.go`](../service/oauth.go). |
| [#466](https://github.com/lejianwen/rustdesk-api/issues/466) | Feature Request: Increase username field length limit from 32 to 64 characters | **Implemented.** [`http/request/api/user.go`](../http/request/api/user.go) and [`http/request/admin/user.go`](../http/request/admin/user.go) use `lte=64`, checked by [`http/request/admin/user_compat_test.go`](../http/request/admin/user_compat_test.go). |
| [#459](https://github.com/lejianwen/rustdesk-api/issues/459) | [Suggestion] Add group-based access control for device connections | **External RustDesk server/client owner.** Enforcement needs an hbbs connection-handshake authorization hook; the API cannot enforce connections alone. |
| [#453](https://github.com/lejianwen/rustdesk-api/issues/453) | 密码无法重置 | **Deferred/unimplemented.** No OIDC-aware password-reset behavior or client integration regression is added. |
| [#429](https://github.com/lejianwen/rustdesk-api/issues/429) | 安卓端不关闭APP，用户登录状态不会失效 | **Partial.** [`http/middleware/rustauth.go`](../http/middleware/rustauth.go) rejects expired/disabled credentials, covered by [`http/middleware/rustauth_compat_test.go`](../http/middleware/rustauth_compat_test.go); already-connected Android clients require RustDesk client/server enforcement. |
| [#427](https://github.com/lejianwen/rustdesk-api/issues/427) | 安全漏洞 /_admin/static | **Implemented.** The legacy admin `StaticFS` tree is absent; [`TestWebInitDoesNotExposeLegacyAdminStaticTree`](../http/router/router_test.go) guards against `/_admin` route registration. |
| [#426](https://github.com/lejianwen/rustdesk-api/issues/426) | logout user client rustdesk windows | **External RustDesk server/client owner.** Remote Windows-client logout requires protocol support. |
| [#424](https://github.com/lejianwen/rustdesk-api/issues/424) | 请问API如何备份？ | **Implemented.** [`docs/operations.md`](operations.md), [`lib/orm/sqlite_backup.go`](../lib/orm/sqlite_backup.go), and [`lib/orm/sqlite_test.go`](../lib/orm/sqlite_test.go) provide and test SQLite snapshots. |
| [#423](https://github.com/lejianwen/rustdesk-api/issues/423) | RUSTDESK_API_RUSTDESK_PERSONAL的作用 | **Support/docs.** [`conf/config.yaml`](../conf/config.yaml) and [`config/rustdesk.go`](../config/rustdesk.go) define/map `rustdesk.personal`; the README documents its environment-variable form. |
| [#422](https://github.com/lejianwen/rustdesk-api/issues/422) | 这两个环境变量哪个是对的? | **Support/docs.** [`conf/config.yaml`](../conf/config.yaml) and [`config/rustdesk.go`](../config/rustdesk.go) define/map canonical `rustdesk.key-file`; deployment examples need reconciliation, not runtime feature work. |
| [#419](https://github.com/lejianwen/rustdesk-api/issues/419) | 需求：添加OTP实现双因素认证 | **Deferred/unimplemented.** No OTP implementation exists. |
| [#400](https://github.com/lejianwen/rustdesk-api/issues/400) | 关于支持别名过滤的问题 | **Implemented.** [`http/controller/admin/peer.go`](../http/controller/admin/peer.go) and [`http/controller/admin/my/peer.go`](../http/controller/admin/my/peer.go) filter aliases, covered by [`http/controller/admin/my/peer_compat_test.go`](../http/controller/admin/my/peer_compat_test.go). |
| [#372](https://github.com/lejianwen/rustdesk-api/issues/372) | Add user-group on AD User | **Deferred/unimplemented.** No verified LDAP user-group admission change or regression is added. |
| [#368](https://github.com/lejianwen/rustdesk-api/issues/368) | 能修改移动端后台页面适配吗？ | **Implemented and browser-verified for the built-in UI.** The login, list, configuration, and management layouts were checked at 320, 768, 1024, and 1440 pixels with no document overflow; a physical mobile-device gate remains separate. |
| [#299](https://github.com/lejianwen/rustdesk-api/issues/299) | 可否考虑API端口与管理网页端口分开。 | **Deferred/unimplemented.** No separate listener/port configuration is implemented. |
| [#165](https://github.com/lejianwen/rustdesk-api/issues/165) | [bug] Device Group | **Deferred/unimplemented.** No offline-device group-membership regression or behavior change is present. |

## Secure connection or RustDesk server integration gate

| Issue | Title | Disposition |
|---:|---|---|
| [#531](https://github.com/lejianwen/rustdesk-api/issues/531) | 我用了s6版本，前面做了一个反向代理nginx，web上https能登陆到后台，但是客户端登陆就报错 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#524](https://github.com/lejianwen/rustdesk-api/issues/524) | 1.48新客户端是不是又无法连接了 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#519](https://github.com/lejianwen/rustdesk-api/issues/519) | 已连接，等待画面传输 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#517](https://github.com/lejianwen/rustdesk-api/issues/517) | 无法登入 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#514](https://github.com/lejianwen/rustdesk-api/issues/514) | 客户端登录账户后，无法连接目标。但是不登录账户时可以使用ID连接 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#512](https://github.com/lejianwen/rustdesk-api/issues/512) | key 不匹配 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#510](https://github.com/lejianwen/rustdesk-api/issues/510) | 突然无法远程，需要退出、重新登录才能远控 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#504](https://github.com/lejianwen/rustdesk-api/issues/504) | MUST-LOGIN无法选择，有没有大佬帮着给看看怎么解决 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#496](https://github.com/lejianwen/rustdesk-api/issues/496) | Cant connect to clients if im logged in | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#486](https://github.com/lejianwen/rustdesk-api/issues/486) | 客户端一直卡在“正在接入 RustDesk 网络...” ，请教 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#485](https://github.com/lejianwen/rustdesk-api/issues/485) | 不支持1.4.4版本 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#484](https://github.com/lejianwen/rustdesk-api/issues/484) | 客户端上不显示设备在线，正常是小绿点，现在是红色的，点击设备也能正常链接，用的S6镜像，官方客户端1.3.6 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#483](https://github.com/lejianwen/rustdesk-api/issues/483) | 连接错误 Failed to secure tcp: Signature mismatch in key exchange: 请稍后再试 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#482](https://github.com/lejianwen/rustdesk-api/issues/482) | Rustdesk Client Version >=1.4.1 no work. | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#480](https://github.com/lejianwen/rustdesk-api/issues/480) | windows客户端和服务端为什么双端都启用websocket时，连接极不稳定…… | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#478](https://github.com/lejianwen/rustdesk-api/issues/478) | Failed to set IP of ID and Relay Server on different IP address | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#455](https://github.com/lejianwen/rustdesk-api/issues/455) | Failed to secure tcp: Handshake failed: invalid public key from rendezvous server: 请稍后再试 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#449](https://github.com/lejianwen/rustdesk-api/issues/449) | 未自定key的情况下使用api报错 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#441](https://github.com/lejianwen/rustdesk-api/issues/441) | S6镜像如何自定义key，rustdesk的参数-k用不了 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#440](https://github.com/lejianwen/rustdesk-api/issues/440) | 如何自定义key，rustdesk的参数-k用不了 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#439](https://github.com/lejianwen/rustdesk-api/issues/439) | 客户端掉登录频繁 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#438](https://github.com/lejianwen/rustdesk-api/issues/438) | handle_punch_hole_request 中如何获取发起控制的peer id | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#437](https://github.com/lejianwen/rustdesk-api/issues/437) | 【Question】server端使用自定义key后，客户端发起连接tab上提示连接不安全，这个怎么解？大家有遇到过吗？ | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#434](https://github.com/lejianwen/rustdesk-api/issues/434) | 1.4.2客户端提示 未就绪,请检查网络连接 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#433](https://github.com/lejianwen/rustdesk-api/issues/433) | Websocket client | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#428](https://github.com/lejianwen/rustdesk-api/issues/428) | 配置多中继服务器 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#421](https://github.com/lejianwen/rustdesk-api/issues/421) | 纯内网通过这个api中转好像会慢半拍再连接到目标 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#420](https://github.com/lejianwen/rustdesk-api/issues/420) | PC端无法连接API server | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#414](https://github.com/lejianwen/rustdesk-api/issues/414) | 如何配置多台中转服务器呢 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#413](https://github.com/lejianwen/rustdesk-api/issues/413) | RustDesk即使设置了固定密码，仍然提示密码错误 | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#382](https://github.com/lejianwen/rustdesk-api/issues/382) | [Research] RustDesk + RustDeskAPI + WebSockets + Reverse HaProxy | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#346](https://github.com/lejianwen/rustdesk-api/issues/346) | 升级1.4.1 API是不是出问题了？ | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |
| [#92](https://github.com/lejianwen/rustdesk-api/issues/92) | 关于PC端链接超时或者链接不上的问题以及解决方案 \| About the problem of timeout or connection failure on PC and how to solve it  | API contract is tested, but connection success requires compatible hbbs secure-TCP/token support; no key-verification bypass. |

## Legacy RustDesk web client - not a supported surface

The new HTMX admin contains no remote-control client code. Legacy bundled web-client artifacts are removed, and [`http/router/router.go`](../http/router/router.go) plus [`http/router/api.go`](../http/router/api.go) do not serve its routes; [`TestLegacyWebClientRoutesAreNeverRegistered`](../http/router/router_test.go) verifies route non-registration. No physical or live deployment verification is implied.

| Issue | Title | Disposition |
|---:|---|---|
| [#506](https://github.com/lejianwen/rustdesk-api/issues/506) | webclient/打开后是这样，只有图没有字 | Unsupported legacy remote-client behavior. Routes and bundled assets are removed (see #500). |
| [#495](https://github.com/lejianwen/rustdesk-api/issues/495) | Webclient error 404 | Unsupported legacy remote-client behavior. Routes and bundled assets are removed (see #500). |
| [#475](https://github.com/lejianwen/rustdesk-api/issues/475) | 配置Oauth管理进行登录和发起webclient出现这种报错是哪里的问题啊 | Unsupported legacy remote-client behavior. Routes and bundled assets are removed (see #500). |
| [#431](https://github.com/lejianwen/rustdesk-api/issues/431) | Web Client 分享 share_token生成加密逻辑 | Unsupported legacy remote-client behavior. Routes and bundled assets are removed (see #500). |
| [#415](https://github.com/lejianwen/rustdesk-api/issues/415) | 因为DMCA，现已将Webclient v2删除 | Unsupported legacy remote-client behavior. Routes and bundled assets are removed (see #500). |
| [#407](https://github.com/lejianwen/rustdesk-api/issues/407) | 二次开发，如何打包全镜像文件 | Unsupported legacy remote-client behavior. Routes and bundled assets are removed (see #500). |
| [#406](https://github.com/lejianwen/rustdesk-api/issues/406) | 目前webclient支持查看摄像头吗，需要怎么配置 | Unsupported legacy remote-client behavior. Routes and bundled assets are removed (see #500). |
| [#402](https://github.com/lejianwen/rustdesk-api/issues/402) | 大佬，你好，我部署好了docker s6镜像的容器后，webclient不起作用 | Unsupported legacy remote-client behavior. Routes and bundled assets are removed (see #500). |
| [#351](https://github.com/lejianwen/rustdesk-api/issues/351) | 发现一个问题，当客户端没有外网时，不能使用WEB客户端。虽然WEB客户端已打包在DOCKER镜像中，但是客户端首次打开时会外联下载一些依赖文件。这个问题是否有修复计划或者有些解决办法呢？ | Unsupported legacy remote-client behavior. Routes and bundled assets are removed (see #500). |
| [#186](https://github.com/lejianwen/rustdesk-api/issues/186) | WebClient的自动连接时，填入的ID的英文大小写有问题。 | Unsupported legacy remote-client behavior. Routes and bundled assets are removed (see #500). |

## RustDesk client, server packaging, or deployment owned

| Issue | Title | Disposition |
|---:|---|---|
| [#525](https://github.com/lejianwen/rustdesk-api/issues/525) | 是否支持绑定多个IP | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |
| [#520](https://github.com/lejianwen/rustdesk-api/issues/520) | 大佬，考虑出个Openwrt的安装包不？ | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |
| [#511](https://github.com/lejianwen/rustdesk-api/issues/511) | 可以自定义ID吗 | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |
| [#492](https://github.com/lejianwen/rustdesk-api/issues/492) | MacBook intel 芯片， 无法开启摄像头和麦 | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |
| [#487](https://github.com/lejianwen/rustdesk-api/issues/487) | 请问怎么自定义KEY，变量设置了无效 | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |
| [#468](https://github.com/lejianwen/rustdesk-api/issues/468) | 网页端登录地址，能原生支持HTTPS么， | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |
| [#452](https://github.com/lejianwen/rustdesk-api/issues/452) | Win7客户端https支持问题 | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |
| [#451](https://github.com/lejianwen/rustdesk-api/issues/451) | 群晖docker运行的非局域网IP怎么处理 | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |
| [#450](https://github.com/lejianwen/rustdesk-api/issues/450) | 防火墙检测到CVE-2007-1156漏洞 | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |
| [#446](https://github.com/lejianwen/rustdesk-api/issues/446) | 如何让客户端更新走国内加速镜像地址？ | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |
| [#442](https://github.com/lejianwen/rustdesk-api/issues/442) | rustdesk 客户端 mac 版本无法打开 | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |
| [#435](https://github.com/lejianwen/rustdesk-api/issues/435) | 电脑被动了 | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |
| [#425](https://github.com/lejianwen/rustdesk-api/issues/425) | apimian.exe无法在windows server 2008 R2中运行 | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |
| [#417](https://github.com/lejianwen/rustdesk-api/issues/417) | v1.4.2 Universal APK for Android | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |
| [#403](https://github.com/lejianwen/rustdesk-api/issues/403) | 有没有大佬自己编译客户端的? | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |
| [#390](https://github.com/lejianwen/rustdesk-api/issues/390) | 可以在服务端控制文件传输权限吗 | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |
| [#353](https://github.com/lejianwen/rustdesk-api/issues/353) | webroot | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |
| [#142](https://github.com/lejianwen/rustdesk-api/issues/142) | [bug] 新版 s6 镜像无法启动 hbbs | Owned by RustDesk client/server, OS packaging, proxy/TLS, or network deployment; documented as an external gate. |

## Support, status, or recovery guidance

| Issue | Title | Disposition |
|---:|---|---|
| [#528](https://github.com/lejianwen/rustdesk-api/issues/528) | 怎么不继续更新了？ | Answered through maintenance, backup/recovery, or configuration guidance; not represented as a product feature. |
| [#523](https://github.com/lejianwen/rustdesk-api/issues/523) | 发现一个新项目 | Answered through maintenance, backup/recovery, or configuration guidance; not represented as a product feature. |
| [#518](https://github.com/lejianwen/rustdesk-api/issues/518) | 刚刚发现一个更新比较勤奋的API仓库 | Answered through maintenance, backup/recovery, or configuration guidance; not represented as a product feature. |
| [#515](https://github.com/lejianwen/rustdesk-api/issues/515) | Is this project dead? | Answered through maintenance, backup/recovery, or configuration guidance; not represented as a product feature. |
| [#513](https://github.com/lejianwen/rustdesk-api/issues/513) | 忘记密码怎么办 | Answered through maintenance, backup/recovery, or configuration guidance; not represented as a product feature. |
| [#507](https://github.com/lejianwen/rustdesk-api/issues/507) | 请问这个项目还会继续维护么？ | Answered through maintenance, backup/recovery, or configuration guidance; not represented as a product feature. |
| [#497](https://github.com/lejianwen/rustdesk-api/issues/497) | 更新维护？ | Answered through maintenance, backup/recovery, or configuration guidance; not represented as a product feature. |
| [#491](https://github.com/lejianwen/rustdesk-api/issues/491) | 大家有没有遇到地址簿某些终端丢失的情况 | Answered through maintenance, backup/recovery, or configuration guidance; not represented as a product feature. |
| [#470](https://github.com/lejianwen/rustdesk-api/issues/470) | what's the admin default password? | Answered through maintenance, backup/recovery, or configuration guidance; not represented as a product feature. |
| [#416](https://github.com/lejianwen/rustdesk-api/issues/416) | 有个bug 尝试提交到官方的时候提交不了 | Answered through maintenance, backup/recovery, or configuration guidance; not represented as a product feature. |
| [#395](https://github.com/lejianwen/rustdesk-api/issues/395) | window部署后，默认密码在哪里查看?另外改成不需要密码登录后，页面能打开，但是不能操作 | Answered through maintenance, backup/recovery, or configuration guidance; not represented as a product feature. |

## Closed issues

All 327 closed issues are listed with their exact state, title, update date, and URL in [upstream-issue-index.md](upstream-issue-index.md). They remain covered by inherited behavior and the route/response regression suite; a closed issue is reopened in this fork only when current official RustDesk evidence demonstrates a regression.
