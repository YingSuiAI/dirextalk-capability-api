# Dirextalk Capability API

独立的、中立的 Capability 协议定义，作为 `dirextalk-agent` 和 `dirextalk-message-server` 的共同依赖。

## 协议版本

- **当前版本**: v1.0.1
- **包名**: `dirextalk.capability.v1`
- **协议**: gRPC + Protobuf

## 目录结构

```
api/proto/dirextalk/capability/v1/  # Protobuf 定义
├── agent_capability.proto          # AgentCapabilityService
├── product_capability.proto        # ProductCapabilityService
├── common.proto                    # 共享类型
└── descriptor.proto                # Capability descriptor

conformance/                        # 一致性测试向量
docs/                              # 协议文档
buf.yaml                           # Buf 配置
buf.gen.yaml                       # 代码生成配置
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
    github.com/YingSuiAI/dirextalk-capability-api v1.0.1
)
```

禁止使用 `replace`、submodule 或分支 SHA。

## 开发

```bash
# 使用仓库脚本（Buf CLI 与 protoc 插件版本均已固定）
bash scripts/generate-go.sh

# Lint
buf lint

# 初始协议的 breaking change 检查（以 main 基线为准）
buf breaking --against '.git#branch=main'

# 生成代码（不要依赖本机缓存的未跟踪 gen/ 文件）
bash scripts/generate-go.sh

# 生成一次性的 mTLS、方向 token 与 Ed25519 grant key fixture
# （目录必须由调用方指定，默认不要写入仓库）
tmp_certs="$(mktemp -d)"
bash scripts/generate-test-certs.sh "$tmp_certs"
```

输出中的 `grant-private.key` 是 message-server 独占的 64 字节签名密钥；
`grant-public.key` 是提供给 Agent 与 Product 验证端的 32 字节公钥。不得把私钥
挂载进 Agent、Product、Flutter 或镜像层。

## 协议原则

1. **中立性**: 不偏向 Agent 或 Product，双向对等
2. **安全优先**: 未知 capability/operation/audience 全部拒绝
3. **确定性**: 使用 RFC 8785 canonical JSON，避免 Struct 精度损失
4. **可追溯**: 每个请求绑定 operation_id、chain_id、digest
5. **防御深度**: mTLS + token + SNI + instance/generation 多层验证

Product delegation 使用私有 `ExchangeProductDelegation`：请求必须显式标注
`QUERY` 或 `START_OPERATION`；Query 不携带 child UUID，Start 的 child UUID 会被
message-server 签入标准 `grant-v1`，Product 在执行边界再次校验。
