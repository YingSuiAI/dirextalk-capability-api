# Capability API 协议文档

## 概述

Dirextalk Capability API 定义了 `dirextalk-agent` 和 `dirextalk-message-server` 之间的双向 gRPC 协议。

## 核心原则

### 1. 双向对等

两个方向使用完全相同的 RPC 结构：

- **AgentCapabilityService**: message-server → Agent
- **ProductCapabilityService**: Agent → message-server

### 2. 结构性防止循环调用

通过以下机制防止死锁：

- **CallContext**: 每个请求携带 `chain_id`、`hop`、`route`
- **最大嵌套深度**: hop ≤ 2
- **编译期依赖方向**: ProductCapability handler 不能注入 Agent client
- **独立连接池**: 每个方向使用独立的 goroutine pool 和 semaphore

允许的调用路径：

```
1. Flutter → AgentCapability
2. Flutter → AgentCapability → ProductCapability
3. Agent background Task → ProductCapability
4. /mcp → ProductCapability
5. /mcp → AgentCapability → ProductCapability (仅 external_mcp audience)
```

禁止：ProductCapability 同步回调 AgentCapability

### 3. 权限模型

- **PermissionContext**: 由 message-server 验证后填充
- **Scopes**: 细粒度权限，如 `contacts:read`, `messages:send`
- **Capability Grant**: 短期、不可篡改的授权令牌
- **Account Generation**: 检测账号删除/重建

### 4. 幂等性

- **operation_id**: UUID，客户端生成并持久化
- **request_digest**: SHA-256(protocol + capability + version + schema_digest + RFC8785(business_input))
- **Receipt**: 终端状态保留 30 天，之后压缩为 tombstone
- **Conflict**: 同 ID + 不同 digest → 永久冲突

### 5. 不确定状态处理

- **Uncertain**: 超时或网络分区后的状态
- **Reconcile**: 服务端协调，最多 5 分钟
- **Side-effect Fence**: uncertain 状态禁止重发副作用

## RPC 说明

### DescribeCapabilities

获取所有可用的 capabilities 及其 operations。

- **何时调用**: 启动时、定期刷新
- **返回**: CapabilityDescriptor 列表、catalog_digest

### Query

执行无副作用的读取操作。

- **特点**: 可安全重试、无幂等要求
- **超时**: 10 秒
- **示例**: 获取联系人列表、查询房间信息

### StartOperation

启动有副作用的 mutation 或 durable stream。

- **幂等**: 同 operation_id + 同 digest → 返回同一结果
- **Revision**: 支持乐观锁
- **超时**: admission 5 秒，业务执行由 operation 自己管理

### GetOperation

获取 operation 当前状态。

- **用途**: 轮询、超时后查询
- **返回**: state + result（如果已完成）+ 最新 sequence

### WatchOperation

监听 operation 事件流。

- **Events**: accepted、progress、result、error、cancelled、gap
- **Resume**: 通过 `after_sequence` 恢复
- **Heartbeat**: 15 秒，45 秒无 heartbeat 重连

### CancelOperation

取消运行中的 operation。

- **语义**: 尽力而为，已执行的副作用无法撤销
- **状态**: 成功取消 → CANCELLED，已完成 → 保持原状态

### ReconcileOperation

协调 uncertain 状态。

- **触发**: 客户端在 uncertain 时调用一次
- **服务端**: 轮询上游、检查副作用、更新状态
- **超时**: 10 秒

## Canonical JSON

使用 RFC 8785 确保确定性：

- 键按字典序排序
- 数字使用精确表示（避免 float64 精度损失）
- 无空白字符
- Unicode 规范化

## 错误码

| Code | 说明 | 可重试 |
|------|------|--------|
| INVALID_ARGUMENT | 参数错误 | 否 |
| TRUST_FAILED | mTLS/token 验证失败 | 否 |
| PERMISSION_DENIED | 权限不足 | 否 |
| NOT_FOUND | operation/capability 不存在 | 否 |
| CONFLICT | digest 冲突 | 否 |
| PRECONDITION_FAILED | revision 不匹配 | 否 |
| NOT_READY | capability 未就绪 | 是（稍后） |
| INCOMPATIBLE | 版本/schema 不兼容 | 否 |
| UNAVAILABLE | 服务不可用 | 是 |
| UNCERTAIN | 不确定状态 | 特殊处理 |
| UPSTREAM_FAILED | 上游失败 | 否 |
| CYCLE_DETECTED | 循环调用 | 否 |
| RESOURCE_EXHAUSTED | 资源耗尽 | 是（稍后） |

## 版本兼容性

- **Protocol Version**: Protobuf 字段编号和 RPC 兼容性
- **Semantic Version**: Capability 业务语义版本
- **Schema Digest**: 精确匹配，不匹配则拒绝

部署时必须验证：

```
Agent protocol_version ∈ [MS.min_protocol, MS.max_protocol]
MS protocol_version ∈ [Agent.min_protocol, Agent.max_protocol]
```

## 安全边界

1. **mTLS**: 双向证书验证
2. **Token**: 独立方向 token 文件
3. **SNI**: 严格 SNI 校验
4. **Instance ID**: 绑定部署实例
5. **Generation**: 账号生成标识
6. **Audience**: 只允许声明的调用方
7. **Schema**: digest 不匹配拒绝
8. **Size**: 超过 max_request_size_bytes 拒绝

## 性能考虑

- **连接池**: MS→Agent 32+64, Agent→MS 64+16
- **超时**: connect 3s, admission 5s, query 10s
- **并发**: 独立 semaphore，排队超过 1s → RESOURCE_EXHAUSTED
- **Backpressure**: Watch 使用 gRPC 流控

## 测试要求

### P0 (阻断发布)

- [ ] Buf lint/breaking 通过
- [ ] Golden digest 测试
- [ ] mTLS/token 拒绝测试
- [ ] Cycle detection 测试
- [ ] 幂等性测试（同 ID 同/不同 digest）
- [ ] Uncertain/reconcile 测试

### P1 (发布后 2 周)

- [ ] RFC 8785 fuzz 测试
- [ ] Schema validation fuzz
- [ ] 并发压力测试

### P2 (持续)

- [ ] 长期 soak 测试
- [ ] Chaos engineering
