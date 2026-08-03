-- TASK-061 秒级 UsageLedger：基线 0001 已建 usage_ledger 表（追加式、
-- 幂等键唯一）；此处补充会话/尝试维度（结算对账）并建查询索引。

ALTER TABLE usage_ledger ADD COLUMN session_id uuid;
ALTER TABLE usage_ledger ADD COLUMN attempt_id uuid;

CREATE INDEX usage_ledger_session_created_idx
    ON usage_ledger (session_id, created_at);

CREATE INDEX usage_ledger_project_round_idx
    ON usage_ledger (project_id, round_sequence);
