#!/usr/bin/env sh
set -o errexit -o nounset

cd "$(dirname "$0")/.."

rm -rf target
mkdir target
echo "Cleaned target/"
