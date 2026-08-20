#!/usr/bin/env sh
set -o errexit -o nounset

. "$(dirname -- "$0")/common.sh"

# Always run what the working tree currently says.
"$ROOT_DIR/bin/build.sh"

exec "$BINARY" "$@"
