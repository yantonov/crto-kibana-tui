# AGENTS.md

## Project
A terminal UI for searching Kibana/OpenSearch logs across Criteo datacenters.
Stack: Go 1.25, Bubble Tea TUI, OpenSearch (no database, no cache).

## Commands
- Run (dev): `make run ARGS="..."`
- Install: `make install`
- Everything else: `make help`
- **Full verification: `make check`**

## Definition of Done
A feature is done only when:
- `make check` is green.
- The feature's specific verification passes.

NOT done: "code is written", "looks right", "tests pass locally", "works on my machine".

## Hard constraints (MUST / MUST NOT)
1. MUST: all credentials stay in the OS keychain (github.com/zalando/go-keyring). No credential files, no env vars for secrets.
2. MUST NOT: commit secrets or access tokens of any kind.
3. MUST NOT: change the `src/` package structure without intent — existing imports depend on the current layout.

## Work rules
- Priority order: correctness → performance → style.
- Match existing Bubble Tea patterns when adding or changing screens.
- Stay inside the task. Mention problems you notice, but don't fix them.

## Escalate to human (stop and ask)
- Changing the config file format (`config.yaml`).
- New external dependencies (new `go.mod` require).
- Any decision affecting credential storage or the keychain.
- Changing the public API (CLI flags, config keys).
- Deleting or renaming `src/` packages.

## Details (read when needed)
- `README.md`: architecture, scripts, credential setup — before touching auth, keychain, or build scripts
- `src/opensearch/`: query builder and search client — before changing search or Elasticsearch queries
- `src/tui/`: Bubble Tea screens (login, filter, results, stats, detail) — before changing any screen
- `src/config/`: config loading and datacenter cache — before changing config keys or datacenters
