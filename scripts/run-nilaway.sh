#!/usr/bin/env bash
# Install (pinned) and run nilaway against a Go module directory.
#
# Usage:
#   scripts/run-nilaway.sh <module-dir> [include-pkgs]
#
# Example:
#   scripts/run-nilaway.sh migration/proxy "github.com/Automaat/finance-buddy/migration/proxy"
#
# Renovate tracks the pinned version below via a customManager (see renovate.json).
set -euo pipefail

# renovate: datasource=go depName=go.uber.org/nilaway
NILAWAY_VERSION="v0.0.0-20260528182042-490362de4fb6"

if [[ $# -lt 1 ]]; then
    echo "Usage: $0 <module-dir> [include-pkgs]" >&2
    exit 2
fi

MODULE_DIR=$1
INCLUDE_PKGS=${2:-}

# Pin the Go toolchain to whatever's on PATH so installing nilaway doesn't
# auto-download an older toolchain (its go.mod has a "go 1.25" directive, and
# without this Go will silently switch to 1.25.x and then fail to analyze our
# 1.26 code).
export GOTOOLCHAIN=local

go install "go.uber.org/nilaway/cmd/nilaway@${NILAWAY_VERSION}"

# Resolve the install location: GOBIN wins if set, else first entry of GOPATH/bin.
# (GOPATH can be a colon-separated list; go install uses its first entry.)
NILAWAY_BIN="$(go env GOBIN)"
if [[ -z "$NILAWAY_BIN" ]]; then
    # ${GOPATH-} avoids tripping `set -u` when GOPATH isn't exported (e.g. CI).
    GOPATH_FIRST="${GOPATH-}"
    GOPATH_FIRST="${GOPATH_FIRST%%:*}"
    if [[ -z "$GOPATH_FIRST" ]]; then
        GOPATH_FIRST="$(go env GOPATH | cut -d: -f1)"
    fi
    NILAWAY_BIN="${GOPATH_FIRST}/bin"
fi
NILAWAY_BIN="${NILAWAY_BIN}/nilaway"

if [[ ! -x "$NILAWAY_BIN" ]]; then
    # Fall back to PATH lookup if the computed path doesn't exist (e.g. when
    # GOBIN points at a directory go install routes elsewhere).
    NILAWAY_BIN="$(command -v nilaway || true)"
    if [[ -z "$NILAWAY_BIN" ]]; then
        echo "Could not locate the nilaway binary after install" >&2
        exit 1
    fi
fi

cd "$MODULE_DIR" || exit 1

if [[ -n "$INCLUDE_PKGS" ]]; then
    "$NILAWAY_BIN" -include-pkgs="$INCLUDE_PKGS" ./...
else
    "$NILAWAY_BIN" ./...
fi
