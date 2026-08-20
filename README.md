# Kibana TUI for crto

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
