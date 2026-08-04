# Dirextalk Capability API

独立的 Capability 协议定义仓库。

## 目标

v1.0.0 协议定义完成，准备集成到 dirextalk-agent 和 dirextalk-message-server。

## 已完成

- [x] Protobuf 定义（common, descriptor, agent_capability, product_capability）
- [x] Buf lint 配置和验证
- [x] Go 代码生成
- [x] 协议文档

## 下一步

1. 创建初始 v1.0.0-dev tag
2. 在 Agent 和 message-server 中集成此 API
3. 实施 mTLS 认证和循环调用防护
4. 实现双向 gRPC 服务

## 关键特性

- 双向对等协议（AgentCapabilityService ⟷ ProductCapabilityService）
- 结构性防止循环调用（CallContext + hop 限制）
- RFC 8785 canonical JSON 保证确定性
- 幂等性支持（operation_id + request_digest）
- 完整的权限模型（PermissionContext + scopes + grants）
