# Dirextalk Capability API

This repository is the neutral contract authority shared by Dirextalk Agent,
Message Server, and owner clients. Keep product implementation and deployment
facts in their owning repositories.

## Change Rules

- Preserve the existing `dirextalk.capability.v1` gRPC wire contract unless a
  request explicitly changes it.
- Treat `api/openapi/agent-data-plane-v2.yaml` and the conformance vectors as
  the authority for the owner-facing Agent HTTP/SSE data plane.
- Generated Go, Dart, and protobuf outputs are checked in. Change their source
  contract first, regenerate with the repository scripts, and verify zero
  drift.
- Never add private keys, bearer tokens, production identifiers, or generated
  certificate fixtures to the repository.
- Use semantic versioning. Compatible contract additions target a minor
  release; do not create tags or publish releases unless explicitly asked.

## Verification

Run the focused contract checks before committing:

```bash
bash scripts/check-generated.sh
go test ./...
go run github.com/bufbuild/buf/cmd/buf@v1.54.0 lint
git fetch --no-tags --force origin refs/heads/main:refs/remotes/origin/main
go run github.com/bufbuild/buf/cmd/buf@v1.54.0 breaking --against '.git#ref=refs/remotes/origin/main'
```
