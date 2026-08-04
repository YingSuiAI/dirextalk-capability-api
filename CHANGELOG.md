# Changelog

## v1.0.0

- 发布中立的 Agent/Product Capability v1 protobuf 与 Go 生成物。
- 固化 RFC 8785 canonical JSON、精确 IEEE-754 数字规则和 32 字节 digest 输入约束。
- 增加 Ed25519 root grant：绑定 chain、root operation/request、owner、generation、
  entry route/deadline、catalog/schema 与 scopes；Agent/Product 使用公钥验证。
- 增加独立 `control-grant-v1` operation control grant，限制 get/watch/cancel/reconcile
  与短 TTL，阻断跨 grant 域重放。
- 为 `StartOperationResponse` 增加向后兼容的可选 control-grant envelope（最多四个
  唯一 action），供 Product 在首次启动和幂等重放时传递 get/watch/cancel/reconcile
  短期授权；Agent 响应可留空，签名私钥不会下发给 Agent。
- 固化 Start replay renewal：ledger 使用 operation ID + descriptor/principal +
  `RootRequestDigest`，同根业务 digest 可在重新验证当前 grant 后只刷新 control grants；
  terminal receipt/state/result 保持不可变，过期 root grant 必须由 message-server
  重新签发，Agent 不得 mint。
- 增加仅限私有 mTLS 的 `ExchangeProductDelegation`：message-server 先验证 Agent
  parent grant，再按显式 `target_kind` 签发精确绑定 Product descriptor/schema/
  request digest 的 child `grant-v1` 并通过 `product_permission` 返回；Query 不携带
  `child_operation_id`，StartOperation 将 child UUID 写入签名 claims；Agent 只转发，
  Product 拒绝 parent、control、截断 route 或跨 descriptor 的 grant。
- Product child grant 增加签名 purpose/target kind、独立两分钟 TTL 和 child UUID
  绑定；tombstone 保留完整 Start replay principal/descriptor key。
- 固化 operation state transition、terminal immutability、restart→uncertain、reconcile
  与 tombstone digest conflict 语义及 conformance tests。
- 固化私有 route 为 `ms→agent→product` 或 `agent→product`，外部来源不进入 route，
  并增加严格 peer-bound、cycle、截断/reset 与跨 chain 校验。
- 统一 gRPC capability metadata、mTLS fixture 生成和 pinned Buf 代码生成流程。
