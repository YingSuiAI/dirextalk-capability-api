# Dirextalk Capability API

独立的、中立的 Capability 协议定义，作为 `dirextalk-agent` 和 `dirextalk-message-server` 的共同依赖。

## 协议版本

- **当前版本**: v1.0.0-dev
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
    github.com/YingSuiAI/dirextalk-capability-api v1.0.0
)
```

禁止使用 `replace`、submodule 或分支 SHA。

## 开发

```bash
# 安装 Buf
go install github.com/bufbuild/buf/cmd/buf@latest

# Lint
buf lint

# Breaking change 检查
buf breaking --against '.git#tag=v1.0.0'

# 生成代码
buf generate
```

## 协议原则

1. **中立性**: 不偏向 Agent 或 Product，双向对等
2. **安全优先**: 未知 capability/operation/audience 全部拒绝
3. **确定性**: 使用 RFC 8785 canonical JSON，避免 Struct 精度损失
4. **可追溯**: 每个请求绑定 operation_id、chain_id、digest
5. **防御深度**: mTLS + token + SNI + instance/generation 多层验证
