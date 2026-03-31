#!/usr/bin/env sh
set -o errexit -o nounset

cd "$(dirname "$0")/.."

MODULE=$(go list -m)
APP=$(basename "$MODULE")

go build -o target/"$APP" ./src
echo "Built target/$APP"
