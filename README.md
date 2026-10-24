# Kibana TUI for crto

A terminal UI for searching Kibana/OpenSearch logs across Criteo datacenters.

## Commands

- Run (dev): `make run ARGS="..."`
- Build: `make build`
- Test: `make test`
- Install: `make install`
- Uninstall: `make uninstall`
- **Full verification: `make check`**
- Everything else: `make help`

`config.yaml` and `datacenters.yaml` are read from the directory holding the
executable — `target/` when run via `make run`, `~/.local/bin` once installed,
or wherever `--config <path>` points. `install` copies the config alongside
the binary; it keeps an already installed one when the two differ, unless
`FORCE_CONFIG=1`. `datacenters.yaml` is a cache the app refetches when missing,
so it is not copied.

## Features

1. search across all datacenters to simplify use cases like search by trace_id
2. copy single pretty printed entry to the clipboard
3. export search results
4. stats with the histogram to check event frequency

## Credentials

Credentials are stored in the OS keychain. Once a login succeeds the app opens
straight on the search form on the next run; the login form only appears when
there is nothing stored or the stored credentials no longer work.

To forget them, press `ctrl+x` on the login form or run `crtokt --logout`.

Set `CRTOKT_KEYCHAIN_SERVICE` to store them under a different keychain service
name, e.g. to share one pair with another tool.
