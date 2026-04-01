#!/usr/bin/env sh
set -euo pipefail

cd "$(dirname "$0")/.."

MODULE=$(go list -m)
APP=$(basename "$MODULE")

CURRENT_DIR=$(pwd)

ln -s "${CURRENT_DIR}/target/" "${HOME}/bin/${APP}-distr"


