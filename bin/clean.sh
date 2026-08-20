#!/usr/bin/env sh
set -o errexit -o nounset

. "$(dirname -- "$0")/common.sh"

rm -f "$BINARY"

echo "Removed $BINARY"
echo "Kept $TARGET_DIR/config.yaml and datacenters.yaml (local settings, not build output)."
echo "Delete them by hand to start from the template again."
