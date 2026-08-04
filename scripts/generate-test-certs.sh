#!/bin/bash
# 生成测试用 mTLS 证书
# 用于开发和测试环境，生产环境应使用正式 CA

set -e

CERTS_DIR="testdata/certs"
mkdir -p "$CERTS_DIR"
cd "$CERTS_DIR"

echo "==> 生成 CA 根证书"
openssl genrsa -out ca-key.pem 4096
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca-cert.pem \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=Dirextalk Test/CN=Dirextalk Test CA"

echo "==> 生成 Agent 服务端证书"
openssl genrsa -out agent-server-key.pem 2048
openssl req -new -key agent-server-key.pem -out agent-server.csr \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=Dirextalk/CN=dirextalk-agent"

# 创建 SAN 配置（支持 localhost 和 127.0.0.1）
cat > agent-server-san.cnf <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req

[req_distinguished_name]

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = dirextalk-agent
IP.1 = 127.0.0.1
EOF

openssl x509 -req -in agent-server.csr -CA ca-cert.pem -CAkey ca-key.pem \
  -CAcreateserial -out agent-server-cert.pem -days 365 \
  -extensions v3_req -extfile agent-server-san.cnf

echo "==> 生成 message-server 客户端证书（访问 Agent）"
openssl genrsa -out ms-client-key.pem 2048
openssl req -new -key ms-client-key.pem -out ms-client.csr \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=Dirextalk/CN=message-server-client"

openssl x509 -req -in ms-client.csr -CA ca-cert.pem -CAkey ca-key.pem \
  -CAcreateserial -out ms-client-cert.pem -days 365

echo "==> 生成 message-server 服务端证书"
openssl genrsa -out ms-server-key.pem 2048
openssl req -new -key ms-server-key.pem -out ms-server.csr \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=Dirextalk/CN=dirextalk-message-server"

cat > ms-server-san.cnf <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req

[req_distinguished_name]

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = dirextalk-message-server
IP.1 = 127.0.0.1
EOF

openssl x509 -req -in ms-server.csr -CA ca-cert.pem -CAkey ca-key.pem \
  -CAcreateserial -out ms-server-cert.pem -days 365 \
  -extensions v3_req -extfile ms-server-san.cnf

echo "==> 生成 Agent 客户端证书（访问 message-server）"
openssl genrsa -out agent-client-key.pem 2048
openssl req -new -key agent-client-key.pem -out agent-client.csr \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=Dirextalk/CN=agent-client"

openssl x509 -req -in agent-client.csr -CA ca-cert.pem -CAkey ca-key.pem \
  -CAcreateserial -out agent-client-cert.pem -days 365

echo "==> 生成方向 token"
# MS→Agent token
openssl rand -hex 32 > ms-to-agent.token
# Agent→MS token
openssl rand -hex 32 > agent-to-ms.token

echo "==> 清理中间文件"
rm -f *.csr *.cnf *.srl

echo "==> 验证证书"
openssl verify -CAfile ca-cert.pem agent-server-cert.pem
openssl verify -CAfile ca-cert.pem ms-server-cert.pem
openssl verify -CAfile ca-cert.pem ms-client-cert.pem
openssl verify -CAfile ca-cert.pem agent-client-cert.pem

echo ""
echo "证书生成完成！"
echo ""
echo "文件清单："
echo "  ca-cert.pem              - CA 根证书（双方都需要）"
echo "  agent-server-cert.pem    - Agent 服务端证书"
echo "  agent-server-key.pem     - Agent 服务端私钥"
echo "  ms-server-cert.pem       - MS 服务端证书"
echo "  ms-server-key.pem        - MS 服务端私钥"
echo "  ms-client-cert.pem       - MS 客户端证书（访问 Agent）"
echo "  ms-client-key.pem        - MS 客户端私钥"
echo "  agent-client-cert.pem    - Agent 客户端证书（访问 MS）"
echo "  agent-client-key.pem     - Agent 客户端私钥"
echo "  ms-to-agent.token        - MS→Agent 方向 token"
echo "  agent-to-ms.token        - Agent→MS 方向 token"
echo ""
echo "⚠️  这些是测试证书，生产环境请使用正式 CA！"
