# SQLite operations

The default SQLite database is `./data/rustdeskapi.db`. Persist the whole `data`
directory on durable storage; do not use an ephemeral container filesystem.

SQLite starts with WAL mode, a five-second busy timeout, foreign keys, and FULL
synchronous writes. WAL creates `rustdeskapi.db-wal` and `rustdeskapi.db-shm`
while the service runs, so copying only the database file is not a safe backup.

Create a consistent backup with the running binary:

```sh
./apimain backup ./backups/rustdeskapi-$(date +%F).db
```

The command writes a SQLite snapshot to a temporary sibling file and renames it
only after it succeeds. It supports SQLite only and never deletes an existing
database before the new snapshot is complete. If the source database is
missing, `backup` fails before application initialization; it does not create
and migrate an empty replacement database.

To restore, stop the API, retain the current `data` directory as a rollback
copy, replace `data/rustdeskapi.db` with a verified backup, remove stale
`rustdeskapi.db-wal` and `rustdeskapi.db-shm` files, then start the API.

An empty SQLite file cannot safely be distinguished from a legitimate first
start without a persisted deployment marker. Startup therefore preserves the
existing first-run behavior instead of guessing that an empty file is a
replacement; use a persistent volume and the backup command to prevent that
case.
