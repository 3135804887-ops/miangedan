# 实时事件契约（realtime-events）

| 字段 | 内容 |
|---|---|
| 文档编号 | API-RT-001 |
| 版本 | 0.1.0（草案，待工程评审） |
| 追踪 | PRD-001 "Realtime Turn Sequence"、Analytics Events；US-03；FR-013 ~ FR-020；NFR-005 ~ NFR-010 |
| 一致性锚点 | `docs/domain/INTERVIEW-STATE-MACHINE.md`（状态与恢复）、`docs/api/openapi.yaml`（业务 API）、`docs/domain/BILLING-STATE-MACHINE.md`（计量联动） |

## 1. 目的

定义客户端、实时房间（SFU）、AI 面试代理、面试工作流（Temporal）与评分服务之间的全部实时事件：名称、方向、字段、幂等键、顺序保证与重试策略，使媒体面、AI 编排与控制面在故障与重连下仍能精确一致。

## 2. 范围

- 房间内实时事件（数据通道 / 信令 / 服务间事件流）。
- 会话生命周期、数字人、回合与字幕、打断、工具、评分结果、计时与计量事件。
- 与 PRD Analytics Events（业务分析事件）的关系与边界。

## 3. 非目标

- 不定义 SFU 内部媒体传输细节（码率、编解码协商）——属部署与供应商适配范畴。
- 不重复定义业务 REST 语义（见 openapi.yaml）。
- 不定义分析指标的统计口径（PRD Success Metrics 已冻结事件名，本文只约束载荷红线）。

## 4. 参与者、方向与通道

| 记号 | 参与者 | 说明 |
|---|---|---|
| C | 客户端（Web/PWA） | 浏览器，持有业务 JWT 与短期房间令牌 |
| R | 实时房间 / SFU 及媒体代理 | 音视频、数据通道、字幕转发 |
| A | AI 面试代理（LangGraph 面试官） | 数字人驱动、ASR/TTS 编排、追问决策 |
| W | 面试工作流（Temporal） | 业务状态、证据账本、计时与计量 |
| S | 评分服务 | 独立评分与复核 |

通道约定：

1. `C ↔ R`：WebRTC 数据通道（可靠有序）+ 媒体轨道。
2. `R ↔ A`：媒体代理内部链路（音频帧、字幕、控制消息）。
3. `R/A ↔ W`：区域事件流（按 `session_id` 分区保序，至少一次投递，消费端幂等）。
4. `W ↔ S`：服务调用（带幂等键）。
5. `W → C`：业务结果经推送/轮询（openapi 会话与结果端点）。

## 5. 通用信封

所有事件携带统一信封：

| 字段 | 类型 | 说明 |
|---|---|---|
| `event_id` | uuid | 全局唯一，投递去重依据 |
| `event_name` | string | 本文定义的点分名称 |
| `session_id` | uuid | 所属会话（分区键） |
| `room_seq` | int64 | 房间内单调递增序号（断线恢复游标） |
| `turn_index` | int \| null | 所属回合号（会话级事件为 null） |
| `occurred_at` | date-time | 发生时间（服务端权威时钟） |
| `data_region` | enum | `cn` / `eu` / `intl` |
| `payload` | object | 各事件定义 |

## 6. 顺序、幂等与重试总规则

1. **顺序**：同一 `session_id` 内 `room_seq` 严格单调；回合事件另按 `turn_index` 局部有序。重连后客户端从最后确认的 `room_seq` 续传。
2. **幂等**：所有事件以 `event_id` 去重；写操作类事件（修订、回答提交、计量）另带业务幂等键（如 `revision_id`、`answer_id`、计量周期键），重复投递副作用为 0（NFR-006）。
3. **重试**：
   - `C → R`：发送失败指数退避（初始 200ms，上限 2s，最多 5 次），超限进入重连流程。
   - `R/A → W`：事件流至少一次投递；消费端幂等；积压告警。
   - `W → S`：评分提交带 `idempotency_key`，超时重试 ≤3 次后进入 `scoring_service_failure` 流程。
4. **持久化红线**：`turn.completed` 之前，上一有效回答必须已完成证据账本持久化（NFR-005）。

## 7. 事件定义

### 7.1 会话生命周期

| 事件 | 方向 | 主要载荷 | 幂等键 | 顺序/重试 |
|---|---|---|---|---|
| `session.created` | W→C | `session_id`、`round_sequence`、`kind`（formal/formal_retry/practice）、`room_join_token`（短期）、`token_expires_at` | event_id | 创建会话响应内下发；令牌一次性 |
| `session.pre_check_passed` | C→R | `device_report`（摄像头/麦克风/网络评级）、`accommodations_frozen` | event_id + session_id | 进入 AVATAR_CONNECTING 前置 |
| `session.reconnect.window_started` | R→C,W | `window_seconds: 180`、`last_confirmed_room_seq` | event_id | 用户断线即触发；计时暂停 |
| `session.reconnected` | C→R→W | `client_resume_token`、`from_room_seq` | event_id + session_id + from_room_seq | 3 分钟窗口内有效；恢复到最后已确认回合 |
| `session.reconnect.expired` | R→W→C | `elapsed_seconds` | event_id | 项目 → EVALUATION_INCOMPLETE |
| `session.auth.paused` | R→C,W | `reason: token_refresh_failed` | event_id | 暂停计时；不判失败 |
| `session.resumed` | C→R→W | `reauth_method` | event_id | 恢复 LIVE |
| `session.ended` | W→各方 | `end_reason`（completed/user_exit/unrecoverable/downgrade_rejected）、`billable_seconds_total` | event_id + session_id + end_reason | 终态事件；触发计量结算 |

### 7.2 数字人（Avatar）

| 事件 | 方向 | 主要载荷 | 说明 |
|---|---|---|---|
| `avatar.connecting` | R→C,W | `provider_ref`（匿名化）、`attempt` | 建连开始（NFR-007：95% ≤8s） |
| `avatar.ready` | R→C,W | `video_track_id`、`audio_track_id`、`character_id` | 进入 LIVE；触发 `usage.meter.started` |
| `avatar.failed` | R→C,W | `failed_component`（video/audio/both）、`error_code`、`retry_attempt` | 持续失败 → PAUSED_SYSTEM；暂停计时、保存状态、自动重连 |
| `avatar.recovered` | R→C,W | `recovered_components` | 恢复 LIVE，恢复计时 |
| `avatar.unrecoverable` | R→C,W | `attempts_made`、`last_error_code` | 自动重连仍失败 |
| `avatar.downgrade_prompted` | R→C | `prompt_id`、`options`（text_interview/end） | 询问是否降级为文字面试；同时发分析事件 `avatar_downgrade_prompted` |
| `avatar.downgrade_accepted` | C→R→W | `prompt_id`、`consent_confirmed: true` | 进入 TEXT_DEGRADED；故障点起不计数字人额度；写 `text_downgrade_accepted` |
| `avatar.downgrade_rejected` | C→R→W | `prompt_id` | 会话结束；项目 → EVALUATION_INCOMPLETE；系统责任全额返还 |

以上事件幂等键均为 `event_id`；`avatar.ready` 与计量事件联动见 7.7。顺序：同会话内按 `room_seq`；`R→W` 至少一次投递。

### 7.3 回合、字幕与转写

| 事件 | 方向 | 主要载荷 | 幂等键 / 备注 |
|---|---|---|---|
| `turn.started` | A→R→C,W | `turn_index`、`question_id`、`question_kind`（main/followup/backup）、`coverage_ids`、`style_snapshot_ref` | event_id；进入答题回合 |
| `turn.question.played` | A→W | `question_id`、`played_text`（实际播放内容，被打断时仅含已播放部分）、`interrupted` | event_id + question_id；**只有实际播放内容写入正式证据** |
| `caption.avatar` | A→R→C | `utterance_id`、`text_delta`、`is_final` | 数字人实时字幕流；增量序列按 utterance_id 内偏移有序 |
| `asr.partial` | R→C,A | `utterance_id`、`text_delta`、`confidence` | 临时文本，仅展示，不入证据 |
| `asr.final` | R→C,A→W | `utterance_id`、`asr_final_text`、`language` | event_id + utterance_id；P95 ≤1s（NFR-010）；原始 ASR 仅诊断 |
| `transcript.revision.submitted` | C→W | `revision_id`、`utterance_id`、`revised_text` | **业务幂等键 `revision_id`**；仅在下一主问题前接受 |
| `transcript.revision.accepted` | W→C | `revision_id`、`effective_for_scoring: true` | 修订文本成为评分证据；原始 ASR 保留诊断 |
| `transcript.revision.rejected` | W→C | `revision_id`、`reason`（window_closed/conflict） | 窗口关闭后拒绝改写冻结回答 |
| `turn.answer.submitted` | C→R→W | `answer_id`、`modality`（voice/text/tool）、`text_answer`（文字模式）、`tool_event_refs` | **业务幂等键 `answer_id`** |
| `turn.completed` | W→各方 | `turn_index`、`evidence_id`、`frozen: true` | 上一有效回答已持久化（NFR-005）后发出；回合冻结 |

### 7.4 打断

| 事件 | 方向 | 主要载荷 | 说明 |
|---|---|---|---|
| `user.interrupt.voice` | C→R→A | `vad_triggered: true`、`at_ms` | 自然语音打断 |
| `user.interrupt.button` | C→R→A | `control: stop_avatar` | 停止按钮；提交文字同样触发 |
| `avatar.output.stopped` | A→R→C | `stopped_utterance_id`、`played_until_ms` | 数字人停止发声并切换聆听；P95 ≤500ms（NFR-009）；未播放部分不入证据 |

### 7.5 岗位工具

| 事件 | 方向 | 主要载荷 | 说明 |
|---|---|---|---|
| `tool.activated` | C→R→W | `tool_key`（code_editor/whiteboard/case_materials/portfolio）、`preconfig_ref` | 仅允许计划中已配置工具；未配置拒绝 |
| `tool.event` | C→R→W | `tool_key`、`tool_event_id`、`event_type`（edit/run/annotate/submit）、`content_ref`（对象存储引用，非内联大对象） | 业务幂等键 `tool_event_id`；事件入证据账本 |
| `tool.snapshot` | W→C | `tool_key`、`snapshot_ref`、`created_at` | 供报告与复核引用 |

### 7.6 评分与结果

| 事件 | 方向 | 主要载荷 | 说明 |
|---|---|---|---|
| `round.scoring.started` | W→S,C | `scoring_request_id`、`attempt_id` | 提交冻结证据、量表与权重；P95 ≤60s（NFR-013） |
| `round.result.published` | S→W→C | `score_id`、`score_version`、`result_status`（PASS/FAIL/EVALUATION_INCOMPLETE）、`round_total`、`critical_gate_passed` | 通过→祝贺+解锁流程；未通过→阻断；未完成→重试入口；同时发分析事件 `round_result_published` |
| `round.review.completed` | S→W→C | `new_score_id`、`supersedes_score_id`、`changed_dimensions`、`reason_summary` | 正式复核产生新版本；历史版本保留 |

### 7.7 计时与计量

| 事件 | 方向 | 主要载荷 | 幂等规则 |
|---|---|---|---|
| `timer.paused` | R→C,W | `reason`（system_fault/reconnect/auth_paused/downgrade_prompted） | event_id；暂停面试计时 |
| `timer.resumed` | R→C,W | `reason_cleared` | event_id |
| `usage.meter.started` | R→W | `meter_period_id`、`started_at` | **业务幂等键 `meter_period_id`**；重复开启去重 |
| `usage.meter.stopped` | R→W | `meter_period_id`、`stopped_at`、`billable_seconds_delta` | 同上；只计 LIVE 实际秒数 |

## 8. 与业务分析事件（Analytics Events）的关系

- PRD 定义的分析事件（`account_created`、`plan_confirmed`、`round_started`、`round_completed`、`round_result_published`、`report_viewed`、`practice_started`、`formal_retry_started`、`review_requested`、`avatar_downgrade_prompted`、`text_downgrade_accepted`、`consent_changed`、`data_export_requested`、`deletion_requested`、`entitlement_reserved`、`usage_refunded`、`resume_parse_confirmed`、`jd_parse_confirmed`）由 W 在相应业务状态迁移时发出，命名保持 snake_case 不变。
- 分析载荷红线：**不包含**简历正文、完整回答、逐字稿内容或原始媒体；只含匿名技术标识与状态枚举。
- 实时事件（本文点分命名）服务于房间协同与证据链路；分析事件服务于指标统计。同一业务事实可由实时事件触发一条分析事件，二者用途不得混用。

## 9. 安全与隐私

1. 房间令牌短期有效（建议 ≤15 分钟，随会话续约），与业务 JWT 隔离；浏览器不持有供应商密钥。
2. 字幕、转写、回答属用户内容：仅在用户所属数据区内传输与存储；事件载荷不含电话、邮箱、证件、地址、保护属性。
3. 监控与日志使用匿名会话编号与技术指标；禁止把面试正文作为日志标签或追踪属性。
4. 原始音视频默认不产生事件流副本；仅在用户单独明确授权（raw_av_recording）且授权有效时存在受控录制链路。

## 10. 异常处理

| 异常 | 处理 |
|---|---|
| 事件乱序到达 | 消费端按 `room_seq`/`turn_index` 重排缓冲；无法收敛时按最后已确认回合恢复 |
| 重复投递 | `event_id` + 业务幂等键双重去重；副作用为 0 |
| 事件流积压 | 告警并限速非关键事件；证据类事件优先；恢复后重放不丢不重 |
| 修订窗口竞态（修订与下一主问题并发） | 以 `turn.completed` 冻结事件为界；冻结后修订一律 `rejected(window_closed)` |
| 计量事件丢失 | 以会话状态迁移日志为权威对账；差异自动冲正并写 Incident |

## 11. 验证方式

1. 契约测试：每个事件的字段、方向、幂等键有模式化校验（与 CI 事件目录比对）。
2. 状态一致性：事件驱动的状态迁移与 `INTERVIEW-STATE-MACHINE.md` 迁移表逐条对齐。
3. 故障演练：重复投递、乱序、积压、重连续传、计量对账的集成与混沌测试。
4. 时延验证：`asr.final` P95 ≤1s、`avatar.output.stopped` P95 ≤500ms 的压测断言。
