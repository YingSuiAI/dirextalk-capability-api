#!/usr/bin/env bash
set -euo pipefail

# Keep the Buf CLI reproducible as well as the remote protoc plugins pinned in
# buf.gen.yaml. Generated Go sources are checked in and this command is the
# single release/CI regeneration entry point.
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
exec go run github.com/bufbuild/buf/cmd/buf@v1.54.0 generate "$@"
