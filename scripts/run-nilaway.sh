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
NILAWAY_VERSION="v0.0.0-20260515015210-fd187751154f"

if [[ $# -lt 1 ]]; then
    echo "Usage: $0 <module-dir> [include-pkgs]" >&2
    exit 2
fi

MODULE_DIR=$1
INCLUDE_PKGS=${2:-}

go install "go.uber.org/nilaway/cmd/nilaway@${NILAWAY_VERSION}"
NILAWAY_BIN="$(go env GOPATH)/bin/nilaway"

cd "$MODULE_DIR"

if [[ -n "$INCLUDE_PKGS" ]]; then
    "$NILAWAY_BIN" -include-pkgs="$INCLUDE_PKGS" ./...
else
    "$NILAWAY_BIN" ./...
fi
