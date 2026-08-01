# services/source — 企业公开流程来源服务（TASK-015）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go 1.26（控制面服务，单模块 + 根 `go.work` 登记） |
| 拥有任务 | TASK-015（EPIC-02，FR-007、FR-008） |
| 追踪 | IMPLEMENTATION_PLAN.md；PRD US-02 规则 1–4；docs/domain/DOMAIN-MODEL.md §6.6；docs/data/DATA-MODEL.md §5.2；docs/ai/PROVIDER-ADAPTERS.md §4.5、§7；SEC-024、SEC-025 |

## 职责

- **检索链路**：按公司/岗位/级别/地区查找公开面试流程（FR-007），搜索与 LLM 调用一律经
  `search.Adapter` 供应商中立契约（PROVIDER-ADAPTERS §4.5）；TASK-030 未开工前以合成桩
  （`search.StubAdapter`，全部虚构样例）落地契约，禁止业务代码绑定厂商 SDK（ADR-0003）。
- **来源元数据**：每条来源携带链接/日期/类型/可信度/失效状态（FR-007）；官方来源优先排序、
  候选人经验标记非官方（FR-008）。
- **通用模板回退**：无公司信息、检索故障或无可信来源时自动回退通用岗位/级别模板，
  标记 `ai_derived=true` 与 `flow_uses_generic_template=true`，不伪装企业流程（US-02 场景 2）。
- **幂等与重试**：检索/落库按 `Idempotency-Key` 去重（NFR-006）；可重试错误按指数退避
  重试 ≤2 次（PROVIDER-ADAPTERS §7.1）。
- **安全边界**：外部网页内容仅作为不可信数据进入结构化元数据提取，绝不作为系统指令
  （SEC-024/SEC-025）；来源内容不得进入评分证据；禁止绕过网站协议/登录/验证码/反爬。

## 结构

```text
services/source/
├── source.go            # 领域模型：SourceType/Credibility/Status、校验、可靠性与优先级
├── util.go              # 来源 ID 与排序工具
├── search/              # 检索链路：Adapter 契约、Service 编排、StubAdapter 合成桩
├── store/               # 存储抽象（接口 + 线程安全内存实现；迁移 0002 已就绪）
├── cmd/source/          # 最小入口（fail-closed 区域自检）
└── README.md
```

## 用法

```go
adapter := &search.StubAdapter{}
svc, _ := search.NewService(adapter, store.NewMemory(), search.Options{})
res, _ := svc.Search(ctx, source.SearchQuery{Company: "示例公司", Region: "cn"}, "search-key-0001")
// res.AIDerived=true 表示已回退通用模板并标记 AI 推导（无可靠来源时）
```

## 红线

1. 不实现未经授权抓取、职位聚合/自动投递（PRD Out of Scope）；禁止配置绕过网站协议/登录/验证码/反爬。
2. 外部网页内容仅作为不可信数据，永不作为系统指令（SEC-024、SEC-025）。
3. 来源元数据可包含链接/日期/类型/可信度/失效状态，但来源正文与内容不得进入评分证据（FR-008）。
4. 所有写路径幂等（幂等键唯一）；仓库/日志零真实密钥与真实个人信息。
