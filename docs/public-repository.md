# Public repository configuration

Examples use reserved `example.com` hostnames and documentation IP addresses.
Replace them in your local deployment configuration. Do not commit deployment
addresses, device inventories, access links, passwords, API tokens, signing keys,
database dumps or screenshots containing login or host details.

Both Compose examples require these local environment variables:

```dotenv
RUSTDESK_ID_SERVER=rustdesk.example.com:21116
RUSTDESK_RELAY_SERVER=rustdesk.example.com:21117
RUSTDESK_API_ORIGIN=https://rustdesk.example.com
RUSTDESK_PUBLIC_KEY=YOUR_RUSTDESK_PUBLIC_KEY
```

Keep real values in your shell, deployment secret manager or an ignored `.env`
file. `RUSTDESK_PUBLIC_KEY` is the contents of the server's **public** key file;
never supply its private key. Configure database, LDAP, OAuth and JWT secrets
through the existing runtime environment variables or private mounted config.
The application generates the initial administrator password at first startup;
there is no shared administrator password to copy from this repository.

Before publishing changes, run a redacted text secret scan and inspect added
screenshots and archives separately; text scanners do not detect passwords in
images. For example:

```sh
go run github.com/zricethezav/gitleaks/v8@v8.30.1 dir . --redact --no-banner
```

Deleting a value in a new commit does not erase it from Git history, tags,
release archives, forks or existing clones. Revoke or rotate any exposed live
credential. Removing historical copies requires a coordinated history rewrite
and, where necessary, GitHub support for cached or otherwise retained copies.
