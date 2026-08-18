# Dirextalk Capability API

独立、中立的共享合同仓库，作为 `dirextalk-agent`、
`dirextalk-message-server` 与 owner client 的共同依赖。现有 gRPC Capability
协议与 owner-facing Agent HTTP/SSE 数据面在这里分别维护，互不改写语义。

## 协议版本

- **当前已发布版本**: v1.0.3
- **下一兼容版本**: v1.1.0
- **gRPC 包名**: `dirextalk.capability.v1`
- **Agent 数据面合同**: OpenAPI 3.1，合同修订 v2，继续使用 `/agent/v1`

## 目录结构

```
api/proto/dirextalk/capability/v1/  # Protobuf 定义
├── agent_capability.proto          # AgentCapabilityService
├── product_capability.proto        # ProductCapabilityService
├── common.proto                    # 共享类型
└── descriptor.proto                # Capability descriptor

api/openapi/agent-data-plane-v2.yaml         # Agent HTTP/SSE 权威合同
conformance/agent-data-plane/v2/             # 跨端一致性向量
contract/agentdatav2/                         # 合同与向量校验
gen/go/dirextalk/agent/data/v2/               # 生成的 Go DTO
gen/dart/dirextalk_agent_data_v2/             # 独立 Dart package
docs/                                          # 协议文档
buf.yaml                                       # Buf 配置
buf.gen.yaml                                   # Protobuf 代码生成配置
```

## 版本策略

使用语义化版本 (SemVer)：

- **patch**: 文档、生成物或不改变 wire 的修复
- **minor**: 向后兼容的字段、operation 或 descriptor 扩展
- **major**: 字段语义、RPC 或状态机破坏性变化

## 依赖方式

Agent 和 message-server 通过 `go.mod` 固定引用正式 tag：

```go
require (
    github.com/YingSuiAI/dirextalk-capability-api v1.0.3
)
```

禁止使用 `replace`、submodule 或分支 SHA。

## 开发

```bash
# 使用仓库脚本（Buf CLI 与 protoc 插件版本均已固定）
bash scripts/generate-go.sh

# 生成 Agent 数据面 Go/Dart 模型（生成器版本与 JAR SHA-256 均已固定）
bash scripts/generate-models.sh

# 在临时目录重建全部生成物并检查零漂移
bash scripts/check-generated.sh

# Lint
go run github.com/bufbuild/buf/cmd/buf@v1.54.0 lint

# 初始协议的 breaking change 检查（以 main 基线为准）
go run github.com/bufbuild/buf/cmd/buf@v1.54.0 breaking --against '.git#branch=main'

# OpenAPI、conformance vectors、生成模型与既有 gRPC 测试
go test ./...

# 生成一次性的 mTLS、方向 token 与 Ed25519 grant key fixture
# （目录必须由调用方指定，默认不要写入仓库）
tmp_certs="$(mktemp -d)"
bash scripts/generate-test-certs.sh "$tmp_certs"
```

输出中的 `grant-private.key` 是 message-server 独占的 64 字节签名密钥；
`grant-public.key` 是提供给 Agent 与 Product 验证端的 32 字节公钥。不得把私钥
挂载进 Agent、Product、Flutter 或镜像层。

Agent 数据面 v2 合同明确 session response、现有 scope 常量、operation
receipt/snapshot、统一安全错误 envelope，以及通用/Turn SSE envelope。Turn SSE
必须显式携带 `operation_id`、`turn_id`、`conversation_id`，且当前合同要求前两者
相等；消费者不得从其中一个身份合成另一个。此兼容新增不改变 ticket claims、
scope 权限、签名密钥或轮换策略。

## 协议原则

1. **中立性**: 不偏向 Agent 或 Product，双向对等
2. **安全优先**: 未知 capability/operation/audience 全部拒绝
3. **确定性**: 使用 RFC 8785 canonical JSON，避免 Struct 精度损失
4. **可追溯**: 每个请求绑定 operation_id、chain_id、digest
5. **防御深度**: mTLS + token + SNI + instance/generation 多层验证

Product delegation 使用私有 `ExchangeProductDelegation`：请求必须显式标注
`QUERY` 或 `START_OPERATION`；Query 不携带 child UUID，Start 的 child UUID 会被
message-server 签入标准 `grant-v1`，Product 在执行边界再次校验。
