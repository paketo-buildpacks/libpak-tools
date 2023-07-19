#!/usr/bin/env bash
set -eo pipefail

VERSION=$(git describe --tags --exact-match 2>/dev/null || echo "")
if [ -z "$VERSION" ]; then
    VERSION=$(git rev-parse --short HEAD)
fi

echo "$VERSION"