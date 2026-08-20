# Kibana TUI for crto

## Contents

- [Features](#features)
- [Scripts](#scripts)
- [Credentials](#credentials)

## Features

1. search across all datacenters to simplify use cases like search by trace_id
2. copy single pretty printed entry to the clipboard
3. export search results
4. stats with the histogram to check event frequency

## Scripts

`bin/build.sh`, `bin/test.sh`, `bin/clean.sh` and `bin/uninstall.sh` do what
their names say. The rest is less obvious:

- `bin/run.sh [args]` rebuilds before it runs.
- `bin/install.sh` copies the binary *and* `target/config.yaml` into
  `~/.local/bin`; `INSTALL_DIR` moves that destination for install and uninstall.

`config.yaml` and `datacenters.yaml` are read from the directory holding the
executable — `target/` under `bin/run.sh`, `~/.local/bin` once installed, or
wherever `--config <path>` points. That is why `install.sh` copies the config at
all; it keeps an already installed one when the two differ, unless
`FORCE_CONFIG=1`. `datacenters.yaml` is a cache the app refetches when missing,
so it is not copied.

## Credentials

Credentials are stored in the OS keychain. Once a login succeeds the app opens
straight on the search form on the next run; the login form only appears when
there is nothing stored or the stored credentials no longer work.

To forget them, press `ctrl+x` on the login form or run `crtokt --logout`.

Set `CRTOKT_KEYCHAIN_SERVICE` to store them under a different keychain service
name, e.g. to share one pair with another tool.
