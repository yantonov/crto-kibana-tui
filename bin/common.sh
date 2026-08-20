#!/usr/bin/env sh
# Shared setup for the scripts in bin/. Source it, do not run it directly.
#
# Exports: ROOT_DIR, MODULE, APP, TARGET_DIR, BINARY
# Side effect: cd to the repository root.

set -o errexit -o nounset

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT_DIR"

MODULE=$(go list -m)
APP=$(basename "$MODULE")
TARGET_DIR="$ROOT_DIR/target"
BINARY="$TARGET_DIR/$APP"
