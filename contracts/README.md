# contracts — 契约生成产物目录

| 字段 | 内容 |
|---|---|
| 追踪 | TASK-001；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 4.1 节 |
| 源契约 | `docs/api/openapi.yaml`、`docs/api/realtime-events.md`、`ai/schemas/*.schema.json` |

## 规则

1. 本目录只存放由源契约**机器生成**的类型/客户端产物；禁止手工编辑生成物。
2. 生成管线与 diff 校验挂接 TASK-016（首个服务端业务任务）接入 CI 阶段 4；此前契约门禁由 `python tools/validate_docs.py` 直接校验源契约。
3. 生成物必须标注来源版本（源契约文件的 git 提交哈希）。
