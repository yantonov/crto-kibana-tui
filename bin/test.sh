#!/usr/bin/env sh
set -o errexit -o nounset

. "$(dirname -- "$0")/common.sh"

unformatted=$(gofmt -l src)
if [ -n "$unformatted" ]; then
	echo "not gofmt'ed:" >&2
	echo "$unformatted" >&2
	echo "run: gofmt -w src" >&2
	exit 1
fi

go vet ./...
go test ./...
