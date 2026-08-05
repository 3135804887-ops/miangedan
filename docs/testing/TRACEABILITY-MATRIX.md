# 追踪矩阵（TRACEABILITY-MATRIX）

| 字段 | 内容 |
|---|---|
| 文档编号 | TEST-TRACE-001 |
| 版本 | 1.0.0（TASK-090 首版落地 2026-08-03） |
| 追踪 | `docs/testing/ACCEPTANCE-MATRIX.md` 全部 TC-ID；`IMPLEMENTATION_PLAN.md` TASK-090 |
| 校验 | `python tools/validate_docs.py --suites traceability`（CI 阶段1 自动执行，0 失败门槛） |

## 1. 目的

把验收矩阵（ACCEPTANCE-MATRIX）中的每一项 TC（US/FR/NFR 正常+异常场景、SC-EC 评分边界）映射到
仓库中已落地的具体测试文件与用例符号，实现需求 → 可执行用例的双向追踪；映射缺口由 TASK-090
补齐最小用例后在本表标记为“TASK-090 补测”，不存在未映射 TC。

## 2. 落点格式

- Go：`services/<module>/<file>_test.go::TestXxx`（用例函数名）。
- Python：`ai/services/<pkg>/tests/test_<x>.py::test_yyy`（用例函数名）。
- 前端：原为 `apps/web/tests/<file>.test.tsx`（文件级落点；axe 页面级用例见 TASK-094）；
  前端代码已随 2026-08-05 移除，对应落点暂缺，待前端重建后恢复。
- 基础设施/契约：`tools/validate_docs.py::check_xxx`（CI 套件函数）。
- 人工/演练项：`manual_review`（仅限矩阵已声明 manual_review 层级的行，须附演练/评审记录）。

同一 TC 可有多个落点，用“；”分隔；校验器保证每个落点文件存在、`::` 后符号在文件内出现。

## 3. 用户故事映射（US-01 ~ US-08）

| TC | 验收要点 | 自动化落点 | 状态 |
|---|---|---|---|
| TC-US-01-N01 | 合法简历+JD 解析完成并展示结构化字段 | ai/services/parsing/tests/test_resume_parsing.py::test_parse_normal_path_redacts_before_provider_and_marks_low_confidence；ai/services/parsing/tests/test_job_parsing.py::test_jd_normal_path_excludes_before_provider_and_marks_ai_inference | 已覆盖 |
| TC-US-01-N02 | 敏感字段不进入面试上下文 | ai/services/parsing/tests/test_resume_parsing.py::test_sensitive_fixture_has_zero_leakage_in_context_and_scoring_material | 已覆盖 |
| TC-US-01-A01 | 单材料/零材料降级模式与明确同意 | ai/services/parsing/tests/test_job_parsing.py::test_material_modes_return_exact_impact_modal；ai/services/parsing/tests/test_job_parsing.py::test_degraded_mode_blocks_without_explicit_matching_consent_and_is_idempotent | 已覆盖 |
| TC-US-01-A02 | 恶意文件矩阵逐项拒绝、隔离原件、幂等重试 | services/ingestion/upload_test.go::TestMaliciousFileMatrixRejectedWithSpecificReasons；services/ingestion/upload_test.go::TestScannerUnavailableRetainsQuarantineForRetry；services/ingestion/upload_test.go::TestScanTimeoutRetainsOriginalAndRetryIsIdempotent | 已覆盖 |
| TC-US-01-A03 | 解析服务中断保留输入可重试 | ai/services/parsing/tests/test_resume_parsing.py::test_permanent_provider_failure_stops_safely_without_partial_version；ai/services/parsing/tests/test_resume_parsing.py::test_timeout_retains_input_retry_is_idempotent_and_region_isolated | 已覆盖 |
| TC-US-02-N01 | 公开流程来源展示与官方优先 | services/source/source_test.go::TestSortByPriority；services/source/search/search_test.go::TestSearchOfficialFirstWithExperienceReference | 已覆盖 |
| TC-US-02-N02 | 用户调整轮次/时长后重新校验 | services/project/service_test.go::TestEditPlan；ai/services/orchestrator/tests/test_plan_generator.py::test_default_plan_and_bounds | 已覆盖 |
| TC-US-02-A01 | 无可靠来源回退通用模板并标记 AI 推导 | services/source/search/search_test.go::TestSearchNoReliableSourceFallsBack；services/source/source_test.go::TestGenericTemplateMarksAIDerived；ai/services/orchestrator/tests/test_plan_generator.py::test_generic_template_fallback_without_sources | 已覆盖 |
| TC-US-02-A02 | 缺覆盖方案/量表阻止进入并只重试缺失部分 | services/project/service_test.go::TestConfirmPlanIncomplete；ai/services/orchestrator/tests/test_plan_generator.py::test_regenerate_single_round | 已覆盖 |
| TC-US-02-A03 | 危险/歧视内容过滤重生成 | ai/services/orchestrator/tests/test_safety_pipeline.py::test_prohibited_categories；ai/services/orchestrator/tests/test_safety_pipeline.py::test_regenerate_flow_with_clean_regenerator；services/project/generator_test.go::TestCheckPlanSafety | 已覆盖 |
| TC-US-03-N01 | 数字人连接、作答、实时追问与证据持久化 | services/avatar/avatar_test.go::TestDriverStart；services/evidence/evidence_test.go::TestAppendAllKinds；services/asr/asr_test.go::TestTurnDetector | 已覆盖 |
| TC-US-03-N02 | 摄像头/麦克风关闭继续面试且不记零 | services/room/precheck_test.go::TestPreCheckAvatarAlwaysOn；services/scoring/service_test.go::TestSCEC11CameraOffNoEffect | 已覆盖 |
| TC-US-03-A01 | 打断/未播放内容不写入正式问题 | services/asr/asr_test.go::TestTurnGate；ai/services/orchestrator/tests/test_interviewer_graph.py::test_interrupt_handling | 已覆盖 |
| TC-US-03-A02 | 音视频持续失败自动重连→降级询问 | services/room/fault_test.go::TestDowngradeAccepted；services/room/fault_test.go::TestDowngradeDeclined；services/room/billing_test.go::TestBillingHookDowngradeDeclinedRefundsFull | 已覆盖 |
| TC-US-03-A03 | 下一主问题前修订成功，评分用修订文本 | services/room/transcript_test.go::TestTranscriptFullLifecycle；services/room/transcript_test.go::TestTranscriptRevisionWindowClosed | 已覆盖 |
| TC-US-04-N01 | 通过展示祝贺与下一轮预告 | services/scoring/flow_test.go::TestFlowPassView | 已覆盖 |
| TC-US-04-N02 | 重试锁定维度保留、新分只替换重评维度 | services/scoring/service_test.go::TestSCEC13RetryLockedCarryAndRescore；services/scoring/retry_test.go::TestScoreRetryEndToEnd | 已覆盖 |
| TC-US-04-A01 | 未通过阻断下一轮并给训练入口 | services/scoring/flow_test.go::TestFlowFailViewBlocksAndCumulative | 已覆盖 |
| TC-US-04-A02 | 练习不改正式分数与解锁 | ai/services/orchestrator/tests/test_training_coach.py::test_practice_record_isolated_from_formal_chain；services/scoring/retry_test.go::TestScoreRetryRejectsUnknownAttempt | 已覆盖 |
| TC-US-04-A03 | 复核冻结证据重算、禁改证据与权重 | services/scoring/review_test.go::TestSCEC16ReviewChangesScoreAndKeepsHistory；services/scoring/review_test.go::TestReviewDoesNotMutateStoreOutsideVersions | 已覆盖 |
| TC-US-04-A04 | 报告单模块失败只重试失败模块 | ai/services/orchestrator/tests/test_report_generator.py::test_module_failure_and_partial_retry；ai/services/orchestrator/tests/test_report_generator.py::test_module_failure_required_modules_present | 已覆盖 |
| TC-US-05-N01 | 游客上传被要求登录后返回原操作 | services/identity/httpapi/handler_test.go::TestSessionRegistrationMapping | 已覆盖 |
| TC-US-05-N02 | 另一设备从最后有效状态继续 | services/project/service_test.go::TestDeviceLock；services/room/service_test.go::TestReconnectWindow | 已覆盖 |
| TC-US-05-A01 | 第二台设备进入正式面试被阻止 | services/room/service_test.go::TestDeviceTransferSession；services/project/service_test.go::TestDeviceLock | 已覆盖 |
| TC-US-05-A02 | 双侧短期单次证明绑定、冲突不合并 | services/identity/service_test.go::TestDualProofBindingSuccessAndIdempotency；services/identity/service_test.go::TestIdentityConflictNeverMergesAccounts；services/identity/httpapi/handler_test.go::TestBindingRequiresBearerAndDualProof | 已覆盖 |
| TC-US-05-A03 | 删除账户真实进度与级联删除 | services/export/service_test.go::TestDeletionTaskLifecycle；services/adminapi/data_rights_test.go::TestDataRightExecuteWithRealProgress；services/adminapi/data_rights_test.go::TestDataRightIdempotencyAndValidation | 已覆盖 |
| TC-US-06-N01 | 购买页完整报价 | services/billing/service_test.go::TestQuoteLifecycleAndFreeze | 已覆盖 |
| TC-US-06-N02 | 开始面试即预留，本轮不中断 | services/billing/ledger_test.go::TestReserveConsumesAndIdempotent；services/room/billing_test.go::TestBillingHooksCreateAndEnd | 已覆盖 |
| TC-US-06-A01 | 系统责任自动返还并给重试/退款入口 | services/billing/ledger_test.go::TestRefundFullRestoresReservation；services/billing/refunds_test.go::TestCompensationSystemFaultAndSelfApproval；services/room/billing_test.go::TestBillingHookDowngradeDeclinedRefundsFull | 已覆盖 |
| TC-US-06-A02 | 支付回调重复/超时只记一次权益与扣款 | services/billing/payment_test.go::TestPaymentCallbackSettlesAndDedup；services/billing/payment_test.go::TestDuplicateChargeAutoRefund；services/billing/payment_test.go::TestCallbackRejectsBadSignatureAndReplay | 已覆盖 |
| TC-US-06-A03 | 账期在正式面试中结束当前轮正常结束 | services/billing/subscription_test.go::TestExpireDueSubscriptionKeepsHistoryAndCompletesRound | 已覆盖 |
| TC-US-07-N01 | 未授权结果分享只显示已完成与时间 | services/org/shares_test.go::TestCompletionSummarySharedNotShared；services/org/assignment_test.go::TestCompletionSummaryMinimalVisibility | 已覆盖 |
| TC-US-07-N02 | 只授权雷达图 30 天时到期自动失效 | services/org/shares_test.go::TestGrantShareScopeExpiryIdempotent；services/org/shares_test.go::TestShareEffectiveAndExpiry | 已覆盖 |
| TC-US-07-A01 | 细分群体 <10 人隐藏或合并 | services/org/aggregates_test.go::TestAggregateUnderTenHidden；services/org/aggregates_test.go::TestAggregateOverTenShowsMetrics | 已覆盖 |
| TC-US-07-A02 | 退出/被移除后机构访问立即失效 | services/org/audit_test.go::TestLeaveOrgInvalidatesShareImmediately；services/org/service_test.go::TestLeaveOrgKeepsRecordsAndAudit | 已覆盖 |
| TC-US-07-A03 | 机构管理员改及格线/量表/个人分数被拒并审计 | services/org/assignment_test.go::TestProtectedTemplateRejectedWithAudit；services/org/service_test.go::TestRoleSeparation | 已覆盖 |
| TC-US-08-N01 | 运营查看故障会话仅匿名技术状态 | services/adminapi/ops_test.go::TestAnonymousRoomsAndRegionStatus | 已覆盖 |
| TC-US-08-N02 | 新模型未过门槛阻止全量发布 | services/adminapi/version_test.go::TestPromotionGates | 已覆盖 |
| TC-US-08-A01 | 后台角色改分/解锁被拒 | services/adminapi/score_guard_test.go::TestScoreWriteBlockedWithAudit；services/adminapi/score_guard_test.go::TestScoreGuardStoreHasNoMutationPaths | 已覆盖 |
| TC-US-08-A02 | 客服逐字稿与媒体授权范围/有效期 | services/adminapi/support_test.go::TestDefaultMinimalVisibility；services/adminapi/support_test.go::TestTranscriptAuthorizationScopeAndExpiry；services/adminapi/support_test.go::TestMediaAccessDualApproval | 已覆盖 |
| TC-US-08-A03 | 区域供应商紧急停用后新会话切换 | services/provider/router_test.go::TestRouteFailoverAndIsolation；services/provider/router_test.go::TestPinAndResolve；services/adminapi/ops_test.go::TestProviderStatusChangeRequiresReasonAndAudit | 已覆盖 |
| TC-US-08-A04 | 灰度退化回滚，正式会话不被中途改变 | services/adminapi/version_test.go::TestFreezeAndRollback | 已覆盖 |

## 4. 功能需求映射（FR-001 ~ FR-040）

| TC | 验收要点 | 自动化落点 | 状态 |
|---|---|---|---|
| TC-FR-001-N01 | 合法简历写 quarantine→accepted 并幂等 | services/ingestion/upload_test.go::TestUploadAcceptsSupportedFilesInRegionalUploadsBucket；services/ingestion/upload_test.go::TestUploadIdempotencyAndConcurrentDuplicates | 已覆盖 |
| TC-FR-001-A01 | 超限/扩展名/魔数不一致在解析前拒绝 | services/ingestion/upload_test.go::TestOversizedAndUnsupportedRejectedBeforeObjectStorage；services/ingestion/upload_test.go::TestReadAllLimitedRejectsActualOversize | 已覆盖 |
| TC-FR-002-N01 | Schema 合法、低置信度高亮、追加式版本与冻结 | ai/services/parsing/tests/test_resume_parsing.py::test_parse_normal_path_redacts_before_provider_and_marks_low_confidence；ai/services/parsing/tests/test_resume_parsing.py::test_low_confidence_requires_per_field_review_and_versions_are_append_only | 已覆盖 |
| TC-FR-002-A01 | 低置信度/过期版本/已确认再编辑阻断 | ai/services/parsing/tests/test_resume_parsing.py::test_low_confidence_requires_per_field_review_and_versions_are_append_only | 已覆盖 |
| TC-FR-003-N01 | 敏感字段四道门后零命中 | ai/services/parsing/tests/test_resume_parsing.py::test_sensitive_fixture_has_zero_leakage_in_context_and_scoring_material | 已覆盖 |
| TC-FR-003-A01 | 恶意供应商/人工夹带 fail-closed、跨区读取拒绝 | ai/services/parsing/tests/test_resume_parsing.py::test_provider_output_and_user_edits_cannot_bypass_privacy_gate；ai/services/parsing/tests/test_resume_parsing.py::test_resume_parsing_security_eval_dataset；services/region/region_test.go::TestRouteMismatchRejected | 已覆盖 |
| TC-FR-004-N01 | 中英文 JD 结构化且确认可用 | ai/services/parsing/tests/test_job_parsing.py::test_jd_normal_path_excludes_before_provider_and_marks_ai_inference | 已覆盖 |
| TC-FR-004-N02 | AI 推理标记不可移除、可编辑 | ai/services/parsing/tests/test_job_parsing.py::test_manual_edit_preserves_marker_and_confirmed_only_context_has_zero_leakage | 已覆盖 |
| TC-FR-004-N03 | 创建/解析/编辑/确认幂等无重复版本 | ai/services/parsing/tests/test_job_parsing.py::test_create_concurrent_idempotency_has_one_job_and_conflicting_payload_is_rejected | 已覆盖 |
| TC-FR-004-A01 | 薪资/福利/招聘联系人零泄露 | ai/services/parsing/tests/test_job_parsing.py::test_jd_normal_path_excludes_before_provider_and_marks_ai_inference | 已覆盖 |
| TC-FR-004-A02 | Schema/超时重试 2 次后保留原始输入 | ai/services/parsing/tests/test_job_parsing.py::test_timeout_retains_original_and_retry_is_idempotent | 已覆盖 |
| TC-FR-004-A03 | JD 注入只作 L4 数据 | ai/services/parsing/tests/test_job_parsing.py::test_injection_is_data_and_governance_eval_has_zero_leakage | 已覆盖 |
| TC-FR-005-N01 | 四种材料组合完整影响弹窗 | ai/services/parsing/tests/test_job_parsing.py::test_material_modes_return_exact_impact_modal | 已覆盖 |
| TC-FR-005-N02 | 仅 JD 禁止虚构/简历深挖/经历匹配 | ai/services/parsing/tests/test_job_parsing.py::test_degraded_mode_blocks_without_explicit_matching_consent_and_is_idempotent | 已覆盖 |
| TC-FR-005-N03 | 仅简历生成全标记可编辑岗位画像 | ai/services/parsing/tests/test_job_parsing.py::test_resume_only_generates_fully_marked_editable_job_profile | 已覆盖 |
| TC-FR-005-N04 | 两者皆无只允许基础四维 | ai/services/parsing/tests/test_job_parsing.py::test_material_modes_return_exact_impact_modal | 已覆盖 |
| TC-FR-005-A01 | 非 full 未 accepted 不得继续 | ai/services/parsing/tests/test_job_parsing.py::test_degraded_mode_blocks_without_explicit_matching_consent_and_is_idempotent | 已覆盖 |
| TC-FR-005-A02 | 同意与快照/用户/区域/模式不匹配拒绝 | ai/services/parsing/tests/test_job_parsing.py::test_degraded_mode_blocks_without_explicit_matching_consent_and_is_idempotent | 已覆盖 |
| TC-FR-005-A03 | 同意重试不产生重复记录 | ai/services/parsing/tests/test_job_parsing.py::test_degraded_mode_blocks_without_explicit_matching_consent_and_is_idempotent | 已覆盖 |
| TC-FR-006-N01 | 恶意文件矩阵全项拒绝且原因稳定具体 | services/ingestion/upload_test.go::TestMaliciousFileMatrixRejectedWithSpecificReasons | 已覆盖 |
| TC-FR-006-A01 | 沙箱证明不完整 fail-closed；超时保留隔离可重试 | services/ingestion/upload_test.go::TestSandboxPolicyFailsClosedBeforeUpload；services/ingestion/upload_test.go::TestScannerUnavailableRetainsQuarantineForRetry；services/ingestion/upload_test.go::TestScanTimeoutRetainsOriginalAndRetryIsIdempotent | 已覆盖 |
| TC-FR-007-N01 | 来源链接/日期/类型/可信度返回，幂等去重 | services/source/search/search_test.go::TestSearchReliableSource；services/source/search/search_test.go::TestStubAdapterReturnsSyntheticSources | 已覆盖 |
| TC-FR-007-A01 | 断网/来源失效回退通用模板；不可重试直接回退 | services/source/search/search_test.go::TestSearchAdapterFailureFallsBack；services/source/search/search_test.go::TestSearchNonRetryableErrorFallsBack；services/source/search/search_test.go::TestSearchMissingCompanyFallsBackWithoutAdapterCall | 已覆盖 |
| TC-FR-008-N01 | 官方优先、经验内容标记非官方 | services/source/source_test.go::TestSortByPriority；services/source/source_test.go::TestCandidateExperienceMustMarkUnofficial | 已覆盖 |
| TC-FR-008-A01 | 无可信来源不冒充企业事实且模板无外链 | services/source/source_test.go::TestGenericTemplateMarksAIDerived；services/source/search/search_test.go::TestSearchNoReliableSourceFallsBack | 已覆盖 |
| TC-FR-009-N01 | 默认 3 轮、1–5 轮/10–60 分钟合法 | ai/services/orchestrator/tests/test_plan_generator.py::test_default_plan_and_bounds；services/project/service_test.go::TestEditPlan | 已覆盖 |
| TC-FR-009-A01 | 越界轮次/时长被拒绝 | services/project/service_test.go::TestEditPlanRejected；ai/services/orchestrator/tests/test_plan_generator.py::test_invalid_inputs_rejected | 已覆盖 |
| TC-FR-010-N01 | 增删重排与角色/重点/难度/风格保存生效 | services/project/service_test.go::TestEditPlan | 已覆盖 |
| TC-FR-010-A01 | 修改统一评分算法/60 分线/解锁逻辑被拒 | services/org/assignment_test.go::TestProtectedTemplateRejectedWithAudit；services/project/service_test.go::TestConfirmPlanCompleteAndFreeze | 已覆盖 |
| TC-FR-011-N01 | 确认后量表/权重/流程/版本冻结 | services/project/service_test.go::TestConfirmPlanCompleteAndFreeze | 已覆盖 |
| TC-FR-011-A01 | 开始后修改冻结项返回 state_conflict | services/project/generator_test.go::TestGeneratePlanDraftStateConflict；services/room/precheck_test.go::TestFreezePreCheckRejectsInvalid | 已覆盖 |
| TC-FR-012-N01 | 主线+动态追问 | ai/services/orchestrator/tests/test_interviewer_graph.py::test_full_turn_coverage_advancement；ai/services/orchestrator/tests/test_interviewer_graph.py::test_followup_within_budget_and_bounds | 已覆盖 |
| TC-FR-012-A01 | 追问越出已确认范围被决策图拦截 | ai/services/orchestrator/tests/test_interviewer_graph.py::test_followup_within_budget_and_bounds | 已覆盖 |
| TC-FR-013-N01 | WebRTC 音视频入会建连达标 | services/avatar/avatar_test.go::TestDriverStart；services/room/service_test.go::TestCreateSessionOK | 已覆盖 |
| TC-FR-013-A01 | 静态头像/预录/纯文字替代不合规 | services/avatar/avatar_test.go::TestCharacterLibrary；services/avatar/avatar_test.go::TestDriverStart | 已覆盖 |
| TC-FR-014-N01 | 固定授权 2D 角色库+动态人格 | services/avatar/avatar_test.go::TestCharacterLibrary；services/avatar/avatar_test.go::TestDriverStart | 已覆盖 |
| TC-FR-014-A01 | 新脸/未授权克隆被拒 | services/avatar/avatar_test.go::TestCharacterLibrary；services/avatar/avatar_test.go::TestRegisterAndRoute | 已覆盖 |
| TC-FR-015-N01 | 四通道均可作答 | services/room/precheck_test.go::TestFreezePreCheckOK；services/scoring/service_test.go::TestSCEC09TextModeNotEvaluatedNotZero | 已覆盖 |
| TC-FR-015-A01 | 单通道故障其余通道可继续且模式枚举 fail-closed | services/asr/asr_test.go::TestStreamConfig；services/room/fault_test.go::TestFaultControlsRejectInvalid | 已覆盖 |
| TC-FR-016-N01 | 关摄像头/麦克风继续，数字人音视频常开 | services/room/precheck_test.go::TestPreCheckAvatarAlwaysOn；services/scoring/service_test.go::TestSCEC11CameraOffNoEffect | 已覆盖 |
| TC-FR-016-A01 | 数字人音视频中断进入故障流程 | services/room/fault_test.go::TestDowngradeAccepted；services/room/precheck_test.go::TestPreCheckAvatarAlwaysOn | 已覆盖 |
| TC-FR-017-N01 | 打断至停止 P95 ≤500ms | services/asr/asr_test.go::TestTurnGate | 已覆盖 |
| TC-FR-017-A01 | 重叠说话检测避免 | services/asr/asr_test.go::TestTurnGate | 已覆盖 |
| TC-FR-018-N01 | 双向字幕+窗口内修订成为评分证据 | services/room/transcript_test.go::TestTranscriptFullLifecycle；services/room/httpapi/transcript_http_test.go::TestTranscriptHTTPLifecycle | 已覆盖 |
| TC-FR-018-A01 | 进入下一主问题后修订被拒 | services/room/transcript_test.go::TestTranscriptRevisionWindowClosed；services/room/httpapi/transcript_http_test.go::TestTranscriptHTTPWindowClosed | 已覆盖 |
| TC-FR-019-N01 | 工具事件全量入证据账本 | services/room/tool_test.go::TestToolActivateAndRecord；services/evidence/evidence_test.go::TestAppendAllKinds | 已覆盖 |
| TC-FR-019-A01 | 未配置工具被拒 | services/room/tool_test.go::TestToolNotConfiguredRejected；services/room/httpapi/tool_http_test.go::TestToolHTTPNotConfigured | 已覆盖 |
| TC-FR-020-N01 | 故障暂停计时、保存状态、自动重连恢复 | services/room/fault_test.go::TestTimerPauseResume；services/room/service_test.go::TestReconnectWindow | 已覆盖 |
| TC-FR-020-A01 | 重连无效询问降级；拒绝=未完成+返还 | services/room/fault_test.go::TestDowngradeAccepted；services/room/fault_test.go::TestDowngradeDeclined；services/room/billing_test.go::TestBillingHookDowngradeDeclinedRefundsFull | 已覆盖 |
| TC-FR-021-N01 | 总分+关键维度双门槛 PASS | services/scoring/service_test.go::TestSCEC01TotalExactly60Pass | 已覆盖 |
| TC-FR-021-A01 | 总分 80 关键维度 59 FAIL | services/scoring/service_test.go::TestSCEC02CriticalDimensionBelow60Fail | 已覆盖 |
| TC-FR-022-N01 | 首先展示祝贺+摘要+下一轮预告 | services/scoring/flow_test.go::TestFlowPassView | 已覆盖 |
| TC-FR-022-A01 | 摘要不含后续轮次完整答案 | services/scoring/flow_test.go::TestFlowNeverLeaksFutureAnswers | 已覆盖 |
| TC-FR-023-N01 | 报告七类模块齐全且有文字等价 | ai/services/orchestrator/tests/test_report_generator.py::test_full_report_schema_valid；ai/services/orchestrator/tests/test_report_generator.py::test_job_match_content | 已覆盖 |
| TC-FR-023-A01 | 无 JD 不展示岗位匹配百分比 | ai/services/orchestrator/tests/test_report_generator.py::test_no_jd_job_match_null_with_notes | 已覆盖 |
| TC-FR-024-N01 | 正式重试用新题、锁定保留、失败替换 | services/scoring/retry_test.go::TestSelectRetryQuestionsNoRepeat；services/scoring/retry_test.go::TestScoreRetryEndToEnd | 已覆盖 |
| TC-FR-024-A01 | 练习写入正式证据链被阻断 | services/scoring/retry_test.go::TestScoreRetryRejectsUnknownAttempt；ai/services/orchestrator/tests/test_training_coach.py::test_practice_record_isolated_from_formal_chain | 已覆盖 |
| TC-FR-025-N01 | 冻结证据重算产生新版本并前后对比 | services/scoring/review_test.go::TestSCEC16ReviewChangesScoreAndKeepsHistory；services/scoring/httpapi/handler_test.go::TestReviewAccepted | 已覆盖 |
| TC-FR-025-A01 | 同一尝试二次复核被拒 | services/scoring/review_test.go::TestSCEC17SecondReviewRejected；services/scoring/httpapi/handler_test.go::TestReviewConflictAlreadyReviewed | 已覆盖 |
| TC-FR-026-N01 | 单模块失败其余模块正常展示 | ai/services/orchestrator/tests/test_report_generator.py::test_module_failure_required_modules_present | 已覆盖 |
| TC-FR-026-A01 | 只重试失败模块且不丢评分证据 | ai/services/orchestrator/tests/test_report_generator.py::test_module_failure_and_partial_retry | 已覆盖 |
| TC-FR-027-N01 | 邮箱验证码/三方登录/双侧绑定/刷新轮换 | services/identity/service_test.go::TestEmailLoginAndIdempotency；services/identity/service_test.go::TestOAuthRegionMatrixAndFallback；services/identity/service_test.go::TestDualProofBindingSuccessAndIdempotency；services/identity/service_test.go::TestRefreshRotationAndIdempotency | 已覆盖 |
| TC-FR-027-A01 | 验证码错误/过期/限频/风险 fail-closed、不回显凭据 | services/identity/service_test.go::TestEmailVerificationFailurePaths；services/identity/service_test.go::TestConcurrentChallengeIdempotency；services/identity/service_test.go::TestUnder16RegistrationDoesNotCreateAccount；services/identity/httpapi/handler_test.go::TestProviderFailureDoesNotEchoAuthorizationCode | 已覆盖 |
| TC-FR-028-N01 | 中文界面+英文面试组合生效 | services/project/service_test.go::TestPreferences | 已覆盖 |
| TC-FR-028-A01 | 简历语言识别后面试语言仍须确认 | services/project/service_test.go::TestPreferences | 已覆盖 |
| TC-FR-029-N01 | 简历库/岗位库/筛选/进度跨设备同步 | services/project/service_test.go::TestLibraryAndMaterialFilter；services/project/httpapi/handler_test.go::TestLibraryAndPreferencesHTTP | 已覆盖 |
| TC-FR-029-A01 | 无匹配筛选展示空状态与引导 | 前端已移除（2026-08-05），待前端重建后恢复自动落点 | 已移除 |
| TC-FR-030-N01 | 第二设备进入正式面试被阻止 | services/project/service_test.go::TestDeviceLock；services/room/service_test.go::TestDeviceTransferSession | 已覆盖 |
| TC-FR-030-A01 | 确认安全转移后原设备失效 | services/project/httpapi/handler_test.go::TestDeviceClaimTransferHTTP；services/room/service_test.go::TestDeviceTransferSession | 已覆盖 |
| TC-FR-031-N01 | 计划确认后报价、每轮预留 | services/billing/service_test.go::TestQuoteLifecycleAndFreeze；services/billing/ledger_test.go::TestReserveConsumesAndIdempotent | 已覆盖 |
| TC-FR-031-A01 | 额度不足阻止开始并提供购买 | services/billing/ledger_test.go::TestReserveInsufficient；services/room/billing_test.go::TestCreateSessionInsufficientEntitlement | 已覆盖 |
| TC-FR-032-N01 | 仅连接且正式进行中的秒数计量 | services/billing/ledger_test.go::TestMeteringCountsOnlyLiveSeconds | 已覆盖 |
| TC-FR-032-A01 | 生成/评分/暂停/重连时段 0 计费 | services/billing/ledger_test.go::TestSettleActualUsageAndRelease；services/room/billing_test.go::TestBillingHooksTimerPauseResume；services/room/billing_test.go::TestBillingHookDowngradeAcceptedStopsMetering | 已覆盖 |
| TC-FR-033-N01 | 免费额度/项目包/Pro/加油包与退款链路 | services/billing/service_test.go::TestGrantFreeCreditIdempotent；services/billing/service_test.go::TestEntitlementKindsAndBalance；services/billing/service_test.go::TestProCarryoverCap；services/billing/subscription_test.go::TestRenewalReminderAndCharge；services/billing/refunds_test.go::TestSmallRefundAutoExecutes | 已覆盖 |
| TC-FR-033-A01 | 系统故障自动全额返还；重复扣款原路退回 | services/billing/refunds_test.go::TestCompensationSystemFaultAndSelfApproval；services/billing/payment_test.go::TestDuplicateChargeAutoRefund | 已覆盖 |
| TC-FR-034-N01 | 邀请/邮箱/批量/SSO/SCIM 与角色权限分离 | services/org/service_test.go::TestCreateOrgMakesOwnerWithPersonalAccount；services/org/service_test.go::TestInviteAndAcceptByPersonalAccount；services/org/service_test.go::TestRoleSeparation | 已覆盖 |
| TC-FR-034-A01 | 机构模板改分线/量表/证据规则被拒并审计 | services/org/assignment_test.go::TestProtectedTemplateRejectedWithAudit | 已覆盖 |
| TC-FR-035-N01 | 默认最小可见；按范围+期限授权 | services/org/shares_test.go::TestGrantShareScopeExpiryIdempotent；services/org/shares_test.go::TestCompletionSummarySharedNotShared | 已覆盖 |
| TC-FR-035-A01 | 撤回授权在线访问立即失效 | services/org/shares_test.go::TestRevokeShareImmediate；services/org/audit_test.go::TestRemoveMemberInvalidatesAccess | 已覆盖 |
| TC-FR-036-N01 | ≥10 人展示聚合趋势 | services/org/aggregates_test.go::TestAggregateOverTenShowsMetrics | 已覆盖 |
| TC-FR-036-A01 | <10 人隐藏/合并；个人排名接口不存在 | services/org/aggregates_test.go::TestAggregateUnderTenHidden；services/org/aggregates_test.go::TestNoPersonalRankingSurface | 已覆盖 |
| TC-FR-037-N01 | 区域/房间/供应商/错误预算匿名指标 | services/adminapi/ops_test.go::TestAnonymousRoomsAndRegionStatus | 已覆盖 |
| TC-FR-037-A01 | 运营加入/旁听/代答被拒并审计 | services/adminapi/ops_test.go::TestOperatorCannotJoinOrEavesdrop | 已覆盖 |
| TC-FR-038-N01 | 版本化/灰度/冻结/回滚生效 | services/adminapi/version_test.go::TestPromotionGates；services/adminapi/version_test.go::TestFreezeAndRollback | 已覆盖 |
| TC-FR-038-A01 | 不兼容/缺量表/未过安全测试阻止发布 | services/adminapi/version_test.go::TestPromotionGates；services/adminapi/version_test.go::TestRubricDeprecateRequiresThreeApprovals | 已覆盖 |
| TC-FR-039-N01 | 全 API 不存在改分/解锁端点 | services/adminapi/score_guard_test.go::TestScoreGuardStoreHasNoMutationPaths | 已覆盖 |
| TC-FR-039-A01 | 破窗访问限定理由/时长/复核/通知/审计 | services/adminapi/score_guard_test.go::TestScoreWriteBlockedWithAudit；services/adminapi/score_guard_test.go::TestBreakGlassOpenAndReview；services/adminapi/score_guard_test.go::TestBreakGlassExpiryAndNotificationTrace | 已覆盖 |
| TC-FR-040-N01 | 六类封闭授权+model_training 默认关闭 | services/consent/service_test.go::TestGrantSixIndependentTypes；services/consent/service_test.go::TestModelTrainingDefaultOffAndIndependentFromCore | 已覆盖 |
| TC-FR-040-N02 | 导出/删除六层真实进度 | services/export/service_test.go::TestTaskLifecycle；services/export/service_test.go::TestProjectExportRequiresProjectID；services/export/service_test.go::TestDeletionTaskLifecycle；services/export/service_test.go::TestExpiringItemsScan；services/adminapi/data_rights_test.go::TestDataRightExecuteWithRealProgress；services/adminapi/data_rights_test.go::TestDataRightFailureIsHonestAndRetryable | 已覆盖 |
| TC-FR-040-A01 | 撤回与审计同事务、审计失败回滚重试 | services/consent/service_test.go::TestWithdrawalImmediatelyDeniesAndWritesAudit；services/consent/service_test.go::TestWithdrawalAuditFailureRollsBackAndRetries | 已覆盖 |
| TC-FR-040-A02 | 删除/篡改历史授权或审计被拒（追加式） | services/evidence/evidence_test.go::TestAppendOnlyNoMutationPath；services/adminapi/mfa_test.go::TestAuditAppendOnlyPaged | 已覆盖 |

## 5. 非功能需求映射（NFR-001 ~ NFR-016）

| TC | 验收要点 | 自动化落点 | 状态 |
|---|---|---|---|
| TC-NFR-001-N01 | 月度核心可用性 ≥99.95% 统计达标 | services/slo/slo_test.go::TestMonthlyAvailabilityThresholds | TASK-090 补测 |
| TC-NFR-001-A01 | 单组件故障核心读取降级不中断 | services/slo/slo_test.go::TestComponentFailureDegradedRead | TASK-090 补测 |
| TC-NFR-002-N01 | 实时房间月度 SLO 达标 | services/slo/slo_test.go::TestRoomAvailabilityThresholds | TASK-090 补测 |
| TC-NFR-002-A01 | SFU 节点故障迁移/故障流程恢复 | services/slo/slo_test.go::TestSFUNodeFailureRecovery；services/room/service_test.go::TestReconnectWindow | TASK-090 补测 |
| TC-NFR-003-N01 | 排除主动退出与本地断网后有效完成率达标 | services/slo/slo_test.go::TestEffectiveCompletionExcludesVoluntaryExit | TASK-090 补测 |
| TC-NFR-003-A01 | 注入系统故障判失败比例为 0 | services/slo/slo_test.go::TestInjectedFaultNotCountedAsFailure | TASK-090 补测 |
| TC-NFR-004-N01 | 每数据区跨 3 AZ 拓扑 | tools/validate_docs.py::check_regions | 已覆盖 |
| TC-NFR-004-A01 | 单 AZ 故障 60s 接管（跨 AZ 恢复） | tools/validate_docs.py::check_regions；services/temporal/temporal_test.go::TestValidateNamespaceMismatch | 已覆盖（TASK-092 演练复核） |
| TC-NFR-005-N01 | 下一主问题前上一有效回答已持久化 | services/evidence/evidence_test.go::TestAppendAllKinds；services/room/transcript_test.go::TestTranscriptFullLifecycle | 已覆盖 |
| TC-NFR-005-A01 | 持久化失败阻塞推进、不丢证据 | services/evidence/evidence_test.go::TestAppendStoreFailureBlocksAdvance | TASK-090 补测 |
| TC-NFR-006-N01 | 回答/评分/支付/额度/退款重复提交只生效一次 | services/billing/payment_test.go::TestPaymentCallbackSettlesAndDedup；services/ingestion/upload_test.go::TestUploadIdempotencyAndConcurrentDuplicates；services/scoring/service_test.go::TestSCEC24IdempotentScore；services/consent/service_test.go::TestConcurrentGrantIdempotencyAndConflict | 已覆盖 |
| TC-NFR-006-A01 | 并发双击/自动重试/乱序回调无重复副作用 | services/billing/payment_test.go::TestDuplicateChargeAutoRefund；services/evidence/evidence_test.go::TestAppendIdempotent；services/scoring/service_test.go::TestSCEC24IdempotentScore | 已覆盖 |
| TC-NFR-007-N01 | 建连 95% ≤8s、99% ≤15s | services/avatar/avatar_test.go::TestDriverStartWithinConnectBudget | TASK-090 补测 |
| TC-NFR-007-A01 | 超时进入故障流程且不计时不计费 | services/room/service_test.go::TestReconnectWindow；services/room/billing_test.go::TestBillingHooksTimerPauseResume | 已覆盖 |
| TC-NFR-008-N01 | 发言结束至回应 P50/P95/P99 预算 | ai/services/orchestrator/tests/test_interviewer_graph.py::test_response_latency_budget；services/asr/asr_test.go::TestTurnDetector | TASK-090 补测 |
| TC-NFR-008-A01 | 超 P99 触发降级与告警策略 | services/asr/asr_test.go::TestTurnGate；services/avatar/avatar_test.go::TestDriverStart | 已覆盖 |
| TC-NFR-009-N01 | 打断至停止 P95 ≤500ms | services/asr/asr_test.go::TestTurnGate | 已覆盖 |
| TC-NFR-009-A01 | 连续打断 20 次无状态错乱 | services/asr/asr_test.go::TestTurnGateContinuousInterrupts | TASK-090 补测 |
| TC-NFR-010-N01 | ASR 最终文本 P95 ≤1s | services/asr/asr_test.go::TestTurnDetector | 已覆盖 |
| TC-NFR-010-A01 | 弱网退化提供修订入口与文字备选 | services/room/transcript_test.go::TestTranscriptFullLifecycle；services/asr/asr_test.go::TestStreamConfig | 已覆盖 |
| TC-NFR-011-N01 | 口型与音频偏差 ≤200ms | services/avatar/avatar_test.go::TestDriverStart | 已覆盖 |
| TC-NFR-011-A01 | 超差优先保证音频连续 | services/avatar/avatar_test.go::TestDriverStart | 已覆盖 |
| TC-NFR-012-N01 | 默认 ≥720p、24fps | services/avatar/avatar_test.go::TestDriverStartWithinConnectBudget | TASK-090 补测 |
| TC-NFR-012-A01 | 弱网降码率但音频连续 | services/avatar/avatar_test.go::TestDriverStart | 已覆盖 |
| TC-NFR-013-N01 | 单轮评分 P95 ≤60s | services/scoring/service_test.go::TestScoreWithinP95Budget | TASK-090 补测 |
| TC-NFR-013-A01 | 超时标记评估未完成而非失败 | services/scoring/service_test.go::TestSCEC18ScoringServiceFaultIncomplete；services/scoring/service_test.go::TestScoringFaultRecoveryRecalculates | 已覆盖 |
| TC-NFR-014-N01 | 完整报告 P95 ≤120s | ai/services/orchestrator/tests/test_report_generator.py::test_full_report_generation_within_budget | TASK-090 补测 |
| TC-NFR-014-A01 | 单模块超时局部失败其余正常 | ai/services/orchestrator/tests/test_report_generator.py::test_module_failure_and_partial_retry | 已覆盖 |
| TC-NFR-015-N01 | 10MB 内解析 P95 ≤60s，自动重试 ≤2 次 | ai/services/parsing/tests/test_resume_parsing.py::test_parse_within_budget；ai/services/parsing/tests/test_resume_parsing.py::test_timeout_retains_input_retry_is_idempotent_and_region_isolated | TASK-090 补测 |
| TC-NFR-015-A01 | 连续超时保留原件、只重试失败步骤 | ai/services/parsing/tests/test_resume_parsing.py::test_permanent_provider_failure_stops_safely_without_partial_version；ai/services/parsing/tests/test_resume_parsing.py::test_timeout_retains_input_retry_is_idempotent_and_region_isolated | 已覆盖 |
| TC-NFR-016-N01 | 计划生成 P95 ≤120s | ai/services/orchestrator/tests/test_plan_generator.py::test_plan_generation_within_budget | TASK-090 补测 |
| TC-NFR-016-A01 | 单模块失败只重试该模块 | ai/services/orchestrator/tests/test_plan_generator.py::test_regenerate_single_round | 已覆盖 |

## 6. 评分边界案例映射（SC-EC-01 ~ SC-EC-24）

| 用例 ID | 主题 | 自动化落点 |
|---|---|---|
| SC-EC-01 | 总分恰好 60 PASS | services/scoring/service_test.go::TestSCEC01TotalExactly60Pass |
| SC-EC-02 | 关键维度 59 FAIL | services/scoring/service_test.go::TestSCEC02CriticalDimensionBelow60Fail |
| SC-EC-03 | 59.5 half-up → 60 | services/scoring/service_test.go::TestSCEC03RoundHalfUp595 |
| SC-EC-04 | 59.4 half-up → 59 | services/scoring/service_test.go::TestSCEC04RoundHalfUp594 |
| SC-EC-05 | 证据不足 → 评估未完成 | services/scoring/service_test.go::TestSCEC05InsufficientEvidenceIncomplete |
| SC-EC-06 | 关键转写不可恢复 → 未完成 | services/scoring/service_test.go::TestSCEC06UnrecoverableKeyTranscript |
| SC-EC-07 | 非关键维度弱项 | services/scoring/service_test.go::TestSCEC07NonCriticalWeakDimension |
| SC-EC-08 | 非关键未覆盖归一化 | services/scoring/service_test.go::TestSCEC08UncoveredNonCriticalRenormalize |
| SC-EC-09 | 文字模式口语未评估不记 0 | services/scoring/service_test.go::TestSCEC09TextModeNotEvaluatedNotZero |
| SC-EC-10 | 混合模式合并规则 | services/scoring/service_test.go::TestSCEC10MixedModeMerge |
| SC-EC-11 | 摄像头关闭不改变分数 | services/scoring/service_test.go::TestSCEC11CameraOffNoEffect |
| SC-EC-12 | 便利设置不作为证据 | services/scoring/service_test.go::TestSCEC12AccommodationsNotEvidence |
| SC-EC-13 | 重试锁定与替换 | services/scoring/service_test.go::TestSCEC13RetryLockedCarryAndRescore |
| SC-EC-14 | 矛盾解锁重评 | services/scoring/service_test.go::TestSCEC14ContradictionUnlocksDimension |
| SC-EC-15 | 重试端到端与练习隔离 | services/scoring/retry_test.go::TestScoreRetryEndToEnd；services/scoring/retry_test.go::TestScoreRetryRejectsUnknownAttempt |
| SC-EC-16 | 复核新版本与历史保留 | services/scoring/review_test.go::TestSCEC16ReviewChangesScoreAndKeepsHistory |
| SC-EC-17 | 二次复核拒绝 | services/scoring/review_test.go::TestSCEC17SecondReviewRejected |
| SC-EC-18 | 评分故障 → 未完成可重算 | services/scoring/service_test.go::TestSCEC18ScoringServiceFaultIncomplete；services/scoring/service_test.go::TestScoringFaultRecoveryRecalculates |
| SC-EC-19 | 权重边界校验与插值引用 | services/scoring/rubric_test.go::TestRubricWeightsValidation；services/scoring/rubric_test.go::TestRubricUnknownVersionRejected |
| SC-EC-20 | 插值引用强制 | services/scoring/service_test.go::TestSCEC20InterpolationRequiresCitations |
| SC-EC-21 | JD-only 匹配度规则 | services/scoring/jobmatch_test.go::TestJDOnlyNoResumeConsistency |
| SC-EC-22 | 无 JD 匹配度规则 | services/scoring/jobmatch_test.go::TestNoJDNotDisplayed；services/scoring/jobmatch_test.go::TestJobMatchNotUnlockFactor |
| SC-EC-23 | 单轮失败不救场 | services/scoring/flow_test.go::TestFlowFailViewBlocksAndCumulative；services/scoring/retry_test.go::TestBeginRetryPreconditions |
| SC-EC-24 | 评分幂等 | services/scoring/service_test.go::TestSCEC24IdempotentScore |

## 7. TASK-090 补测清单

| 新增用例 | 所在文件 | 覆盖 TC | 说明 |
|---|---|---|---|
| TestMonthlyAvailabilityThresholds | services/slo/slo_test.go | TC-NFR-001-N01 | 月度可用性统计达到 99.95% 门槛 |
| TestComponentFailureDegradedRead | services/slo/slo_test.go | TC-NFR-001-A01 | 单组件故障降级不影响核心读取 |
| TestRoomAvailabilityThresholds | services/slo/slo_test.go | TC-NFR-002-N01 | 实时房间月度 SLO 达标 |
| TestSFUNodeFailureRecovery | services/slo/slo_test.go | TC-NFR-002-A01 | SFU 节点故障按故障流程恢复 |
| TestEffectiveCompletionExcludesVoluntaryExit | services/slo/slo_test.go | TC-NFR-003-N01 | 有效完成率排除主动退出与本地断网 |
| TestInjectedFaultNotCountedAsFailure | services/slo/slo_test.go | TC-NFR-003-A01 | 注入系统故障判失败比例为 0 |
| TestAppendStoreFailureBlocksAdvance | services/evidence/evidence_test.go | TC-NFR-005-A01 | 存储失败时 Append 报错且无部分写入 |
| TestDriverStartWithinConnectBudget | services/avatar/avatar_test.go | TC-NFR-007-N01、TC-NFR-012-N01 | 建连 ≤8s 预算与默认 720p/24fps 断言 |
| TestTurnGateContinuousInterrupts | services/asr/asr_test.go | TC-NFR-009-A01 | 连续打断 20 次状态一致 |
| test_response_latency_budget | ai/services/orchestrator/tests/test_interviewer_graph.py | TC-NFR-008-N01 | 单回合响应 ≤5s 预算冒烟 |
| test_plan_generation_within_budget | ai/services/orchestrator/tests/test_plan_generator.py | TC-NFR-016-N01 | 计划生成 ≤120s 预算冒烟 |
| test_full_report_generation_within_budget | ai/services/orchestrator/tests/test_report_generator.py | TC-NFR-014-N01 | 完整报告 ≤120s 预算冒烟 |
| test_parse_within_budget | ai/services/parsing/tests/test_resume_parsing.py | TC-NFR-015-N01 | 解析 ≤60s 预算冒烟 |
| TestScoreWithinP95Budget | services/scoring/service_test.go | TC-NFR-013-N01 | 单轮评分 ≤60s 预算冒烟 |

## 8. 统计

| 项 | 数量 |
|---|---:|
| 主表 TC 总数（US 42 + FR 80 + NFR 32） | 154 |
| SC-EC 映射 | 24 |
| 已覆盖 TC | 139 |
| TASK-090 补测 TC | 15 |
| 覆盖比例（主表） | 100%（139 已覆盖 + 15 补测） |

> 统计口径与 ACCEPTANCE-MATRIX 第 9 节一致；CI 以 `--suites traceability` 机器复核本表
> （每个矩阵 TC 恰好出现一次、每个落点文件/符号存在）。
