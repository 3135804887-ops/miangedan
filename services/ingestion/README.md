# ingestion 服务（材料上传与文件安全）

| 字段 | 内容 |
|---|---|
| 技术基线 | Go（控制面服务，每服务一个模块，经根 `go.work` 统一工作区） |
| 拥有任务 | TASK-012 |
| 追踪 | IMPLEMENTATION_PLAN.md；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 4.1 节 |

## TASK-012 实现

- `Service`：按 `(data_region,user_id,Idempotency-Key)` 创建一次上传；相同键不同内容散列冲突，
  并发重放不重复写对象或扫描（FR-001、NFR-006）。
- `UploadObjectStore`：接口只暴露所属区域 uploads 桶的 quarantine/accepted 操作，无法写
  exports/media；安全拒绝删除隔离副本，扫描超时/暂时不可用保留原件供重试。
- `AttestedSandbox`：运行时必须证明无出站网络、一次性、只读根文件系统、无凭证挂载；
  证明不完整时服务构造即 fail-closed（TM-02、SEC-020）。
- `Scanner`：不信任扩展名或 Content-Type，按魔数与容器结构检测 PDF/DOC/DOCX，逐项拒绝
  病毒、宏、压缩炸弹、伪装、超限、损坏、加密，并返回稳定原因与具体用户说明（FR-006）。
- `MalwareDetector` 为供应商中立接口；`SyntheticSignatureDetector` 只用于
  `fixtures/synthetic/upload-security/manifest.yaml` 对应的确定性合成回归，生产组合必须注入
  经验证的加固恶意软件引擎。

数据库契约见 `services/migrate/migrations/0012_resume_uploads.sql`；API 契约见
`POST /v1/uploads/resumes`、`GET /v1/uploads/{uploadId}`、`POST /v1/uploads/{uploadId}:retry`。
