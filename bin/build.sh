#!/usr/bin/env sh
set -o errexit -o nounset

. "$(dirname -- "$0")/common.sh"

mkdir -p "$TARGET_DIR"

# -trimpath keeps absolute build paths out of the binary.
go build -trimpath -o "$BINARY" ./src

echo "Built $BINARY"
