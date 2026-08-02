# services/asr — 流式语音识别与回合/打断（TASK-022）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go 1.26（共享服务，依赖 services/provider 与 services/region） |
| 拥有任务 | TASK-022（EPIC-03）；真实 ASR 接入随供应商选型 |
| 追踪 | FR-017；NFR-008 ~ NFR-010；docs/ai/PROVIDER-ADAPTERS.md §4.2；docs/api/realtime-events.md |

## 职责

- **流式识别契约**：`Provider.OpenStream` 双向流（发送音频帧 → `partial`/`final` 事件，
  §4.2）；合成桩供开发/测试；`ValidateConfig` 语言/静音断点 fail-closed。
- **回合检测**：`TurnDetector` 静音窗口断点 → final；断点→final 时延预算 `ASRFinalBudget=1s`
  （NFR-010）。
- **打断与防重叠**：`TurnGate` 单说话方闸门（用户说话时数字人不得输出，FR-017 避免重叠说话）；
  语音 VAD/停止按钮触发打断，`StopConfirmed` 校验打断→停止时延预算 `StopLatencyBudget=500ms`
  （NFR-009）。
- **适配层集成**：`RegisterProvider` 注册 `asr_{region}_{role}` 至 TASK-030 注册表
  （版本固定、主备路由 + 熔断）。

## 红线

1. `partial` 仅展示不入证据；`final` 为评分证据输入（realtime-events 契约）。
2. 打断即停：P95 ≤500ms；最终文本 P95 ≤1s（NFR-009/010）。
3. 原始音频不持久化；凭证按区隔离、日志零输出。
