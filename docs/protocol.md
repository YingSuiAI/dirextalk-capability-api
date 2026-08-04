# Capability API 协议文档

## 概述

Dirextalk Capability API 定义了 `dirextalk-agent` 和 `dirextalk-message-server` 之间的双向 gRPC 协议。

## 核心原则

### 1. 双向对等

两个方向使用完全相同的 RPC 结构：

- **AgentCapabilityService**: message-server → Agent
- **ProductCapabilityService**: Agent → message-server

两条私有 gRPC 方向统一使用 metadata：`authorization: DTX-Capability-Token
<token>`、`x-dirextalk-instance-id` 和
`x-dirextalk-account-generation`。不再定义方向各异的 token scheme；token
解析必须使用共享 API 的严格 parser。

### 2. 结构性防止循环调用

通过以下机制防止死锁：

- **CallContext**: 每个请求携带 `chain_id`、`hop`、`route`
- **最大嵌套深度**: hop ≤ 3（允许完整的 `ms→agent→product`）
- **编译期依赖方向**: ProductCapability handler 不能注入 Agent client
- **独立连接池**: 每个方向使用独立的 goroutine pool 和 semaphore
- **运行时 chain fence**: Agent 发起 Product RPC 前把 `chain_id` 标记为
  product-active；该 RPC 返回前，Agent 服务拒绝同一 chain 的任何新入站调用并
  返回 `CYCLE_DETECTED`，退出路径必须可靠清除标记

`CallContext.route` 遵循“发送方追加、接收方校验并前进”的约定：发送方先用
`AppendCallNode` 写入自己的节点，接收方使用
`ValidateAndAdvanceCallContext` 校验相邻拓扑并追加自身节点。因此，完整链路
必须记录为 `ms→agent→product`；Agent 和 Product 的 peer-bound helper 会拒绝
空 route、错误发送方或 Product 后续转发。接收方随后成为发送方时，重复追加
自己的尾节点是幂等 no-op，避免把合法的 `ms→agent→product` 链误判为循环。

外部来源只在 message-server 的 HTTP/WS/MCP 边界完成认证，不写入私有
`CallContext.route`。允许的私有调用路径：

```
1. ms → agent
2. ms → agent → product
3. agent → product（仅携带 message-server 预先签发且仍有效的 delegation）
```

Flutter 等客户端始终只直连 message-server；MCP 外部请求也先进入
message-server，再按上述私有路径执行。

禁止：ProductCapability 同步回调 AgentCapability

### 3. 权限模型

- **PermissionContext**: 由 message-server 验证后填充；`root_request_digest` 是
  grant-independent parent/product business digest，必须与当前签名 grant 的
  对应边界一致，Agent 只转发 opaque child permission，不可自行改写或签名
- **Scopes**: 细粒度权限，如 `contacts:read`, `messages:send`
- **Capability Grant**: message-server 使用 Ed25519 私钥签发的短期授权令牌；
  Agent 与 Product 只持有 32 字节公钥，不能签发 owner/scope 权限
- **Grant purpose**: `grant-v1` 仍使用同一标准 envelope，但签名的
  `grant_kind` 必须是 `root` 或 `product-child`；Product child 还绑定
  `product_target_kind=query|start_operation`。Query child 不带
  `child_operation_id`，Start child 必须绑定嵌套 operation UUID。
- **Account Generation**: 检测账号删除/重建

Grant 的 `root_capability_id/root_operation` 描述当前授权边界。Agent 向
Product 发起嵌套调用前，必须通过私有 `ExchangeProductDelegation` broker 把
message-server 签发的 Agent parent grant 换成一个 Product-bound `grant-v1`：
child 复用 parent 的 chain/root operation、owner、generation 和 scopes，但将
`root_capability_id/root_operation/root_request_digest/catalog_digest/schema_digest`
精确绑定 Product 目标 descriptor 与 canonical business input。Product 只能接受
这个精确 child grant，不能把 `agent.*` parent grant 当作 Product grant；没有
有效 delegation 时，后台任务不得通过过期 grant 调用 Product，当前协议不签发
长期全能 grant。

签名同时绑定 `chain_id`、真实的 root `operation_id`、root request digest 和
最初的 `entry_route/entry_hop`。
Agent 根边界必须精确匹配该入口；Product 边界必须验证实际 CallContext 以该
入口为前缀且是合法的 `agent→product` 下一跳，从而拒绝 Product-bound route
截断。所有 owner、generation、root capability/operation、root request digest、
catalog/schema digest 与所需 scopes 均为必填绑定，不存在“空值表示跳过校验”。
对原始 `route=ms` 的重置重入由 product-active chain fence 拒绝；Product handler
包还必须通过依赖测试证明不能导入/注入 Agent client 或其凭据。Product RPC
完成后的同业务重试只能命中相同 operation ID + `RootRequestDigest` 的幂等账本，
不能创建第二次副作用。最终 `request_digest` 仍必须按本次携带的当前 grant
重新计算并验证；grant 更新只改变授权传输，不改变业务幂等键。
Agent 的 root StartOperation 必须调用 `ValidateRootOperationRequest`，保证请求
`operation_id == call_context.root_operation_id`；Product 子操作只调用普通
`ValidateOperationRequest`，因为其子 operation ID 与签名的 root ID 不同。
Agent Query 必须使用 `VerifyAgentQueryGrant`：除了精确的 `ms→agent` route、
chain/root UUID、owner/generation、RootRequestDigest 和 scopes，还必须逐字节绑定
root capability/operation 以及 catalog/schema digest；不能只验证 grant 非空或只比较
owner。Query 的 `operation_id` 是 descriptor operation 名称，不取代 root UUID。
`RootRequestDigest` 必须先用 `ComputeRootRequestDigest` 对业务输入计算；它不包含
尚未生成的 grant。message-server 签名后，再用 `ComputeRequestDigest`（包含
grant digest）生成最终 `StartOperationRequest.request_digest`，避免循环摘要。

### 4. 幂等性

- **operation_id**: UUID，客户端生成并持久化
- **request_digest**: SHA-256(protocol + capability + version + schema_digest + RFC8785(business_input) + 每个 attachment digest + 当前 signed grant digest)；schema、grant 与每个 attachment digest 必须都是 32 字节。每次 Start replay 都必须基于当前 grant 重新计算和验证该最终摘要。
- **RootRequestDigest**: message-server 签发 grant 前使用的 grant-independent canonical business preimage；要求 schema/attachment digest，但省略尚未存在的 grant digest。operation ledger 的冲突键使用 operation ID + capability/operation + owner/generation + `RootRequestDigest`，而不是最终 grant digest。
- **Receipt**: 终端状态保留 30 天，之后压缩为 tombstone；tombstone 必须保留
  完整 `StartReplayKey`（operation UUID、capability/operation、owner、generation、
  `RootRequestDigest`），缺少任一字段不得接受 replay
- **Conflict**: 同 ID + 不同 root/business digest 或 principal/descriptor → 永久冲突；同 root digest 搭配新的、仍有效且精确绑定的 grant 是允许的控制授权刷新，不得改变已有 receipt/state/result。

### 5. 不确定状态处理

- **Uncertain**: 超时或网络分区后的状态
- **Reconcile**: 服务端协调，最多 5 分钟
- **Side-effect Fence**: uncertain 状态禁止重发副作用

### 6. Operation 状态机与恢复

合法转换为：`PENDING→RUNNING`、`PENDING→CANCELLED`、`RUNNING→COMPLETED`、
`RUNNING→FAILED`、`RUNNING→CANCELLED`，以及任意非终态
`PENDING/RUNNING→UNCERTAIN`（进程重启、上游超时或网络分区时必须这样落盘）。
`UNCERTAIN` 只能通过 `ReconcileOperation` 解析为 `COMPLETED/FAILED/CANCELLED`；
不能直接重放副作用。`COMPLETED/FAILED/CANCELLED` 是终态，结果、错误和
sequence 不可再修改；重复请求只能返回同一 receipt。tombstone 保留
完整 `StartReplayKey`、最终状态、revision 和冲突标记；同 ID 的新
root digest 永久返回 `CONFLICT`，同 root digest 的 grant renewal 仍返回原
receipt，不得重新执行。

### 7. Control Grant

`OperationControlGrant` 使用独立的 `control-grant-v1` 域分离签名，TTL 不得超过
2 分钟。它只绑定一个目标 operation UUID、owner、account generation、入口 route、
deadline、一个动作（`get|watch|cancel|reconcile`）和唯一派生 scope
`operation:control:<action>`。Root/Product grant verifier 必须拒绝 control grant；
control verifier 也必须拒绝 root grant，并逐字段拒绝跨 operation、action、owner、
generation 或过期重放。`VerifyOperationControlGrant` 接收 Agent 实际请求
`CallContext`，内部从 claims 的 `entry_route/entry_hop` 推导并严格要求
`ms→agent`/hop 2；调用者不得先手工 strip 成 `ms`/hop 1。Product 控制面使用
`VerifyProductOperationControlGrant`，严格要求 `ms→agent→product` 或
`agent→product`。

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

### ExchangeProductDelegation (private)

仅供 Agent→message-server 的私有 mTLS broker 使用，Flutter/HTTP/WS 不暴露此
RPC。请求携带已认证的 parent `PermissionContext`、当前 `CallContext`、
`target_kind`、Product descriptor/operation 和 canonical `request_json`。
`QUERY` 的 `child_operation_id` 必须为空；`START_OPERATION` 必须提供新的
嵌套 child UUID。message-server 必须先用 parent 的
`root_request_digest` 精确验证 Agent grant，再按 Product catalog/schema 计算
Product `RootRequestDigest`，使用唯一的 Ed25519 私钥签发短期 Product-bound
`grant-v1`，并通过 `product_permission` 返回。Agent 没有私钥，只能原样转发
`product_permission`。

Product Start/Query 边界必须用 `VerifyProductDelegationGrant` 绑定实际
`ms→agent→product` 或 `agent→product` route、outer root operation、owner、
generation、scopes、Product capability/operation、catalog/schema、Product
`RootRequestDigest` 和 target kind；`START_OPERATION` 还必须逐字节匹配签名的
`child_operation_id` 与请求 operation UUID。parent `agent.*` grant、control grant
或 route 截断均拒绝。最终 `request_digest` 仍以 child grant digest 重新计算，
delegation TTL 不超过 2 分钟；过期后重新走 broker，Agent 不得 mint。

### StartOperation

启动有副作用的 mutation 或 durable stream。

- **幂等**: 同 operation_id + 同 `RootRequestDigest` → 返回同一业务 receipt；最终 digest 每次按当前 grant 重新验证
- **Revision**: 支持乐观锁
- **超时**: admission 5 秒，业务执行由 operation 自己管理
- **Control grants**: Product 可在响应中返回可选的
  `control_grants`。每个 envelope 包含唯一的 `action`（`get`、`watch`、
  `cancel` 或 `reconcile`）、不透明的 `control-grant-v1` grant bytes 以及
  `expires_at_unix_ms`；单个响应最多 4 个，grant 的有效期不超过 2 分钟。
  message-server/客户端应在使用前调用
  `ValidateOperationControlGrantEnvelopes`，随后在对应 control RPC 边界使用
  Ed25519 公钥验证 grant。Agent StartOperation 响应可以留空该列表。
  相同 operation ID + `RootRequestDigest` 的幂等重放必须返回不可变的业务
  receipt/state/result，但可以重新签发一组精确绑定同一 owner、generation、
  operation 的短期 control grants；旧组应立即作废或自然过期，control grants
  不是 receipt 字段。重放不得扩大 action、owner、operation 或有效期。

Root grant 过期后，调用方必须先向 message-server 申请一个新的、同一业务
`RootRequestDigest` 和授权边界的 exact delegation，再重放 StartOperation。Agent
只有公钥，不能 mint 或自行延长 root/control grant；最终 digest 必须绑定新 grant
并在 Product 边界重新校验。

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
- 对象键按 UTF-16 code unit 排序（RFC 8785）
- 数字使用 ECMAScript/RFC 8785 的最短 IEEE-754 表示
- Go 整数与 JSON 整数词法值使用同一规则：必须能被 IEEE-754 binary64
  精确表示（如 `2^53`、`10^17` 可以，`2^53+1`、`10^17+1` 不可以）；
  不能精确表示的业务整数应使用经过 schema 约束的字符串
- 无空白字符
- 不做 Unicode 规范化；输入必须是有效 UTF-8/I-JSON

`request_json` 必须已经是 RFC 8785 canonical JSON，`request_digest` 必须是
32 字节 SHA-256。最终摘要中的 schema、grant 与 attachment digest 均必须是
32 字节；服务端应重新计算摘要并使用常量时间比较；客户端传入的
`PermissionContext` 不能改变 owner、scope 或 account generation，只有
message-server 签发并验证的 grant 才具有权限语义。

`ValidateOperationRequest`/`ValidateQueryRequest` 只做请求形状校验（包括
canonical JSON、digest 长度、权限上下文和 revision）；服务端仍必须基于能力
版本、schema digest、业务输入和 grant 重新计算摘要，再调用
`VerifyRequestDigest`。

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
6. **Grant signer**: Ed25519 私钥只存在于 message-server；Agent 仅接收并转发
   broker 返回的 Product child grant，不能 mint 或延长 delegation；Product 仅公钥
7. **Audience**: 只允许声明的调用方
8. **Schema**: digest 不匹配拒绝
9. **Size**: 超过 max_request_size_bytes 拒绝
10. **结构隔离**: Product handler package 不得导入或持有 Agent client、Agent
    token 或 grant 私钥；该约束与 entry route/chain/Ed25519 签名绑定共同防止
    reset、跨 chain 和伪造回调。

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
- [ ] 幂等性测试（同 ID + 同 root digest 可刷新 controls；不同 root digest 拒绝；最终 receipt/state/result 不变）
- [ ] Root grant 过期后只能由 message-server exact-renewal，Agent 无 mint 权限
- [ ] Uncertain/reconcile 测试

### P1 (发布后 2 周)

- [ ] RFC 8785 fuzz 测试
- [ ] Schema validation fuzz
- [ ] 并发压力测试

### P2 (持续)

- [ ] 长期 soak 测试
- [ ] Chaos engineering
