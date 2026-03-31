#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

MODULE=$(go list -m)
APP=$(basename "$MODULE")

./target/"$APP" --diag --env preprod --app catalogapi-catalog-api --timeframe 1h
