#!/usr/bin/env bash
set -euo pipefail

# Generate disposable local mTLS fixtures. The output directory is explicit so
# callers can keep private keys outside the repository (for example, under a
# freshly-created mktemp directory in CI).
#
# Usage:
#   scripts/generate-test-certs.sh /tmp/dirextalk-capability-certs

if [[ $# -ne 1 || -z "$1" ]]; then
  echo "usage: $0 OUTPUT_DIR" >&2
  exit 2
fi

out_dir="$(readlink -f "$1")"
mkdir -p "$out_dir"
chmod 700 "$out_dir"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/dirextalk-certs.XXXXXX")"
cleanup() {
  rm -rf "$work_dir"
}
trap cleanup EXIT
chmod 700 "$work_dir"

cd "$work_dir"

openssl genrsa -out ca-key.pem 4096 2>/dev/null
openssl req -new -x509 -days 3650 -sha256 -key ca-key.pem -out ca-cert.pem \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=Dirextalk Test/CN=Dirextalk Test CA"

make_cert() {
  local role="$1" cn="$2" eku="$3" san="$4"
  local key="${role}-key.pem" csr="${role}.csr" cert="${role}-cert.pem"
  local cnf="${role}.cnf"

  openssl genrsa -out "$key" 2048 2>/dev/null
  openssl req -new -sha256 -key "$key" -out "$csr" \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=Dirextalk/CN=${cn}"
  cat >"$cnf" <<EOF
[v3_req]
basicConstraints = critical,CA:FALSE
keyUsage = critical,digitalSignature,keyEncipherment
extendedKeyUsage = ${eku}
subjectAltName = ${san}
EOF
  openssl x509 -req -sha256 -in "$csr" -CA ca-cert.pem -CAkey ca-key.pem \
    -CAcreateserial -out "$cert" -days 365 -extensions v3_req -extfile "$cnf"
}

# Server certificates are serverAuth-only; client certificates are
# clientAuth-only. This prevents accidentally using a fixture in the opposite
# direction while still exercising mTLS in both private gRPC directions.
make_cert agent-server dirextalk-agent serverAuth "DNS:localhost,DNS:dirextalk-agent,IP:127.0.0.1"
make_cert ms-server dirextalk-message-server serverAuth "DNS:localhost,DNS:dirextalk-message-server,IP:127.0.0.1"
make_cert ms-client message-server-client clientAuth "DNS:message-server-client"
make_cert agent-client agent-client clientAuth "DNS:agent-client"

# Shared API metadata requires exactly 32 random bytes encoded as unpadded
# base64url (43 ASCII characters).
openssl rand -base64 32 | tr '+/' '-_' | tr -d '=\n' >ms-to-agent.token
openssl rand -base64 32 | tr '+/' '-_' | tr -d '=\n' >agent-to-ms.token

# Only message-server receives the raw 64-byte Ed25519 signing key. Agent and
# Product receive the raw 32-byte public key. OpenSSL's PKCS#8/SPKI wrappers
# end with the seed/public bytes respectively; Go's Ed25519 private form is
# seed || public.
openssl genpkey -algorithm ED25519 -out grant-signing.pem 2>/dev/null
openssl pkey -in grant-signing.pem -outform DER | tail -c 32 >grant-seed.bin
openssl pkey -in grant-signing.pem -pubout -outform DER | tail -c 32 >grant-public.key
cat grant-seed.bin grant-public.key >grant-private.key
test "$(wc -c <grant-private.key)" -eq 64
test "$(wc -c <grant-public.key)" -eq 32

for cert in agent-server-cert.pem ms-server-cert.pem ms-client-cert.pem agent-client-cert.pem; do
  openssl verify -CAfile ca-cert.pem "$cert" >/dev/null
done

cp ca-cert.pem agent-server-cert.pem agent-server-key.pem \
  ms-server-cert.pem ms-server-key.pem ms-client-cert.pem ms-client-key.pem \
  agent-client-cert.pem agent-client-key.pem ms-to-agent.token agent-to-ms.token \
  grant-private.key grant-public.key \
  "$out_dir/"
chmod 600 "$out_dir"/*-key.pem "$out_dir"/*.token "$out_dir/grant-private.key"
chmod 644 "$out_dir"/*-cert.pem "$out_dir/ca-cert.pem" "$out_dir/grant-public.key"
echo "mTLS fixtures written to $out_dir"
