#!/bin/sh
# usage: ./build.sh [debug|release|test]
set -eu

cd "$(dirname "$0")"

git_hash=""
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    git_hash="$(git describe --always --dirty --exclude '*')"
fi
flags=""
ldflags="-X main.BUILD_GIT_HASH=$git_hash"
case "${1:-debug}" in
    debug) ;;
    release) flags="-trimpath" ldflags="$ldflags -s -w -X main.BUILD_DEBUG=false" ;;
    test) exec go test -v ./... ;;
    *)
        echo "usage: $0 [debug|release|test]" >&2
        exit 2
        ;;
esac

go vet ./...
mkdir -p build
CGO_ENABLED=0 go build $flags -ldflags="$ldflags" -o build/git-trend .
