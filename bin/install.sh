#!/usr/bin/env sh
set -euo pipefail

cd "$(dirname "$0")/.."

MODULE=$(go list -m)
APP=$(basename "$MODULE")

CURRENT_DIR=$(pwd)

INSTALL_DIR="${HOME}/.local/bin"

ln -s "${CURRENT_DIR}/target/" "${INSTALL_DIR}/${APP}-distr"

echo "Symlink was created '${INSTALL_DIR}/${APP}-distr'"
