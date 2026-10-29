#!/bin/sh
set -eu

cd "$(dirname "$0")"

# --- Arguments ---------------------------------------------------------------
mode="debug"
action="build"

while [ "$#" -gt 0 ]; do
    case "$1" in
        debug|release) mode="$1" ;;
        build|run|test) action="$1" ;;
        --)
            shift
            break
            ;;
        *)
            echo "usage: $0 [debug|release] [build|run|test] [-- args...]" >&2
            exit 2
            ;;
    esac
    shift
done

if [ "$action" != "run" ] && [ "$#" -gt 0 ]; then
    echo "error: program arguments require the 'run' action" >&2
    exit 2
fi

# --- Actions -----------------------------------------------------------------
case "$action" in
    test)
        exec go test -v -race ./...
        ;;
esac

# --- Version -----------------------------------------------------------------
version="$(git describe --abbrev=0 --tags --always 2>/dev/null || echo unknown)"

# --- Flags -------------------------------------------------------------------
build_debug=""
build_release="-trimpath"

linker_common="-X main.Version=$version"
linker_debug=""
linker_release="-s -w"

# --- Choose Flags ------------------------------------------------------------
case "$mode" in
    debug)
        echo "[debug mode]" >&2
        build_flags="$build_debug"
        linker_flags="$linker_common $linker_debug"
        ;;
    release)
        echo "[release mode]" >&2
        build_flags="$build_release"
        linker_flags="$linker_common $linker_release"
        ;;
esac

# --- Vet ---------------------------------------------------------------------
# Go's version of -Wall, so every build runs it
go vet ./...

# --- Build -------------------------------------------------------------------
mkdir -p build
env CGO_ENABLED=0 go build $build_flags "-ldflags=$linker_flags" -o build/git-trend .

# --- Run ---------------------------------------------------------------------
if [ "$action" = "run" ]; then
    exec ./build/git-trend "$@"
fi
