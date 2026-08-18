#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output_root="$repo_root"
if [[ "${1:-}" == "--output-root" ]]; then
  if [[ $# -ne 2 || -z "$2" ]]; then
    echo "usage: $0 [--output-root DIR]" >&2
    exit 2
  fi
  output_root="$(mkdir -p "$2" && cd "$2" && pwd)"
elif [[ $# -ne 0 ]]; then
  echo "usage: $0 [--output-root DIR]" >&2
  exit 2
fi

openapi_generator_version="7.15.0"
openapi_generator_sha256="4da1a7cdb78c3a43b1eab0648891135e8c3547d2eedba0dd69daf377f865f366"
tool_cache_root="${XDG_CACHE_HOME:-${TMPDIR:-/tmp}/dirextalk-contract-tools}"
generator_dir="$tool_cache_root/openapi-generator/$openapi_generator_version"
generator_jar="$generator_dir/openapi-generator-cli.jar"
mkdir -p "$generator_dir"
if [[ ! -f "$generator_jar" ]] || ! printf '%s  %s\n' "$openapi_generator_sha256" "$generator_jar" | sha256sum --check --status; then
  curl --fail --silent --show-error --location \
    --output "$generator_jar.download" \
    "https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/$openapi_generator_version/openapi-generator-cli-$openapi_generator_version.jar"
  printf '%s  %s\n' "$openapi_generator_sha256" "$generator_jar.download" | sha256sum --check --status
  mv "$generator_jar.download" "$generator_jar"
fi

staging_root="$(mktemp -d)"
trap 'rm -rf -- "$staging_root"' EXIT
contract="$repo_root/api/openapi/agent-data-plane-v2.yaml"

mkdir -p "$staging_root/gen/go/dirextalk/agent/data/v2"
go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 \
  --config "$repo_root/scripts/oapi-codegen-v2.yaml" \
  -o "$staging_root/gen/go/dirextalk/agent/data/v2/models.gen.go" \
  "$contract"

dart_staging="$staging_root/gen/dart/dirextalk_agent_data_v2"
java -jar "$generator_jar" generate \
  --generator-name dart \
  --input-spec "$contract" \
  --output "$dart_staging" \
  --global-property 'models,modelDocs=false,modelTests=false,supportingFiles' \
  --additional-properties 'pubName=dirextalk_agent_data_v2,pubLibrary=dirextalk_agent_data_v2,pubVersion=1.1.0,pubDescription=Generated Dirextalk Agent Data Plane v2 models,pubHomepage=https://github.com/YingSuiAI/dirextalk-capability-api,pubRepository=https://github.com/YingSuiAI/dirextalk-capability-api'

mkdir -p \
  "$output_root/gen/go/dirextalk/agent/data/v2" \
  "$output_root/gen/dart/dirextalk_agent_data_v2"
rsync --archive --delete \
  "$staging_root/gen/go/dirextalk/agent/data/v2/" \
  "$output_root/gen/go/dirextalk/agent/data/v2/"
rsync --archive --delete --delete-excluded \
  --exclude '.openapi-generator/' \
  --exclude '.openapi-generator-ignore' \
  --exclude '.gitignore' \
  --exclude '.travis.yml' \
  --exclude 'git_push.sh' \
  "$dart_staging/" \
  "$output_root/gen/dart/dirextalk_agent_data_v2/"
