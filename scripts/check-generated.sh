#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
scratch="$(mktemp -d)"
trap 'rm -rf -- "$scratch"' EXIT

bash "$repo_root/scripts/generate-models.sh" --output-root "$scratch/models"
diff --recursive --unified \
  "$repo_root/gen/go/dirextalk/agent/data/v2" \
  "$scratch/models/gen/go/dirextalk/agent/data/v2"
diff --recursive --unified --exclude='.dart_tool' --exclude='pubspec.lock' \
  "$repo_root/gen/dart/dirextalk_agent_data_v2" \
  "$scratch/models/gen/dart/dirextalk_agent_data_v2"

bash "$repo_root/scripts/generate-go.sh" --output "$scratch/protobuf"
while IFS= read -r generated; do
  relative="${generated#"$scratch/protobuf/gen/go/"}"
  cmp "$generated" "$repo_root/gen/go/$relative"
done < <(find "$scratch/protobuf/gen/go" -type f -name '*.pb.go' -print | sort)
