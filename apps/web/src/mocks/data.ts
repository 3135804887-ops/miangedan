/**
 * 合成演示数据（Mock_Layer）。
 * 全部标记 synthetic: true，不含任何真实个人信息；驱动页面五态测试。
 */

import { PROJECT_STATUSES, SESSION_STATUSES } from '@mgd/domain-states';

const [
  _DRAFT,
  _PARSING,
  MATERIAL_REVIEW,
  _PARSE_FAILED,
  _PLAN_GENERATING,
  PLAN_REVIEW,
  _PLAN_FAILED,
  READY,
  IN_SESSION,
  _SCORING,
  ROUND_PASSED,
  ROUND_FAILED,
  _PRACTICING,
  _EVALUATION_INCOMPLETE,
  COMPLETED,
] = PROJECT_STATUSES;

const [_ROOM_CREATED, _PRE_CHECK, _AVATAR_CONNECTING, LIVE, _PAUSED_SYSTEM, _RECONNECTING, _DOWNGRADE_PROMPTED, _TEXT_DEGRADED, _AUTH_PAUSED, _ENDED] = SESSION_STATUSES;

export const synthetic = { synthetic: true } as const;

export interface MockProject {
  project_id: string;
  name: string;
  company: string;
  job_title: string;
  status: (typeof PROJECT_STATUSES)[number];
  interview_language: 'zh-CN' | 'en-US';
  current_round_sequence: number;
  total_rounds: number;
  next_action: string;
  created_at: string;
  updated_at: string;
}

export const mockProjects: readonly MockProject[] = [
  {
    project_id: 'p-0001',
    name: '后端工程师面试训练（合成科技）',
    company: '合成科技',
    job_title: '后端工程师',
    status: ROUND_PASSED,
    interview_language: 'zh-CN',
    current_round_sequence: 1,
    total_rounds: 3,
    next_action: '进入第 2 轮：岗位专业面试',
    created_at: '2026-07-28T10:00:00Z',
    updated_at: '2026-08-01T09:30:00Z',
  },
  {
    project_id: 'p-0002',
    name: '产品经理面试训练',
    company: '示例数据公司',
    job_title: '高级产品经理',
    status: READY,
    interview_language: 'zh-CN',
    current_round_sequence: 1,
    total_rounds: 2,
    next_action: '开始第 1 轮',
    created_at: '2026-07-30T03:00:00Z',
    updated_at: '2026-08-01T12:00:00Z',
  },
  {
    project_id: 'p-0003',
    name: '前端工程师（英文面试）',
    company: 'Global Sample Inc.',
    job_title: 'Frontend Engineer',
    status: PLAN_REVIEW,
    interview_language: 'en-US',
    current_round_sequence: 0,
    total_rounds: 3,
    next_action: '确认面试计划',
    created_at: '2026-08-01T01:00:00Z',
    updated_at: '2026-08-01T08:00:00Z',
  },
  {
    project_id: 'p-0004',
    name: '数据分析师（无 JD 降级）',
    company: '',
    job_title: '数据分析师',
    status: MATERIAL_REVIEW,
    interview_language: 'zh-CN',
    current_round_sequence: 0,
    total_rounds: 3,
    next_action: '校对解析结果',
    created_at: '2026-08-01T14:00:00Z',
    updated_at: '2026-08-01T15:00:00Z',
  },
  {
    project_id: 'p-0005',
    name: '架构师模拟面试',
    company: '合成架构实验室',
    job_title: '解决方案架构师',
    status: ROUND_FAILED,
    interview_language: 'zh-CN',
    current_round_sequence: 1,
    total_rounds: 2,
    next_action: '复盘练习后可正式重试',
    created_at: '2026-07-25T06:00:00Z',
    updated_at: '2026-07-31T11:00:00Z',
  },
  {
    project_id: 'p-0006',
    name: '全流程训练（已完成）',
    company: '示例企业',
    job_title: '测试开发工程师',
    status: COMPLETED,
    interview_language: 'zh-CN',
    current_round_sequence: 3,
    total_rounds: 3,
    next_action: '查看完整报告',
    created_at: '2026-07-20T02:00:00Z',
    updated_at: '2026-07-24T10:00:00Z',
  },
];

export interface MockPlanRound {
  sequence: number;
  round_type: string;
  role: string;
  focus: string;
  duration_minutes: number;
  difficulty: 'basic' | 'standard' | 'challenge';
  critical_dimensions: readonly string[];
  tools: readonly string[];
  ready: boolean;
}

export const mockPlan = {
  project_id: 'p-0003',
  plan_version: 1,
  rubric_version: 'rubrics/v1/default',
  frozen: false,
  dimension_weights: {
    professional_competence: 25,
    problem_solving: 20,
    communication: 15,
    experience_evidence: 15,
    behavioral_collaboration: 15,
    learning_adaptability: 10,
  },
  rounds: [
    {
      sequence: 1,
      round_type: 'screening_resume_deepdive',
      role: '招聘角色',
      focus: '围绕简历经历结构化深挖，验证经历一致性',
      duration_minutes: 25,
      difficulty: 'standard',
      critical_dimensions: ['experience_evidence', 'behavioral_collaboration'],
      tools: [],
      ready: true,
    },
    {
      sequence: 2,
      round_type: 'role_professional',
      role: '专业面试官',
      focus: '考察岗位核心能力与问题解决',
      duration_minutes: 30,
      difficulty: 'standard',
      critical_dimensions: ['professional_competence', 'problem_solving'],
      tools: ['code_editor'],
      ready: true,
    },
    {
      sequence: 3,
      round_type: 'comprehensive_final',
      role: '综合面试官',
      focus: '综合评估学习适应性与跨场景行为',
      duration_minutes: 20,
      difficulty: 'challenge',
      critical_dimensions: ['communication', 'learning_adaptability'],
      tools: [],
      ready: false,
    },
  ] satisfies readonly MockPlanRound[],
  round_weights: [
    { sequence: 1, weight: 33 },
    { sequence: 2, weight: 34 },
    { sequence: 3, weight: 33 },
  ],
  flow_uses_generic_template: false,
  quote_minutes: 75,
} as const;

export const mockSession = {
  session_id: 's-0001',
  project_id: 'p-0001',
  round_sequence: 2,
  room_status: LIVE,
  elapsed_seconds: 482,
  billable_seconds: 0,
  paused_seconds: 0,
  current_turn_index: 1,
  reconnect_deadline_at: null,
} as const;

export const mockResult = {
  score_id: 'sc-0001',
  round_sequence: 1,
  result_status: 'PASS',
  round_total: 74,
  pass_line: 60,
  critical_gate_passed: true,
  dimensions: [
    { dimension: 'professional_competence', score: 78, is_critical: true },
    { dimension: 'problem_solving', score: 72, is_critical: true },
    { dimension: 'communication', score: 81, is_critical: false },
    { dimension: 'experience_evidence', score: 76, is_critical: true },
    { dimension: 'behavioral_collaboration', score: 68, is_critical: false },
    { dimension: 'learning_adaptability', score: 70, is_critical: false },
  ],
  strengths: ['结构化表达清晰', '项目复杂度把握准确'],
  attention: ['行为类问题回答偏短，建议补充团队协作实例'],
  next_round: {
    sequence: 2,
    role: '专业面试官',
    focus: '考察岗位核心能力与问题解决',
    difficulty: 'standard',
    duration_minutes: 30,
  },
} as const;

export const mockReport = {
  report_id: 'r-0001',
  project_id: 'p-0006',
  report_kind: 'full',
  project_status: COMPLETED,
  overall: { status: 'PASS', total: 76, passed_rounds: 3 },
  round_trajectory: [
    { round: 1, role: '招聘角色', total: 71, status: 'PASS' },
    { round: 2, role: '专业面试官', total: 78, status: 'PASS' },
    { round: 3, role: '综合面试官', total: 80, status: 'PASS' },
  ],
  job_match: {
    available: true,
    match_percent: 82,
    must_have: ['Go/Python 服务端开发', '数据库设计', '分布式系统基础'],
    plus: ['K8s 经验', '开源贡献'],
  },
  dimensions: [
    { dimension: 'professional_competence', score: 80 },
    { dimension: 'problem_solving', score: 76 },
    { dimension: 'communication', score: 82 },
    { dimension: 'experience_evidence', score: 74 },
    { dimension: 'behavioral_collaboration', score: 72 },
    { dimension: 'learning_adaptability', score: 78 },
  ],
  evidence: [
    {
      question: '请介绍一个你主导的复杂项目',
      answer_summary: '以订单中心重构为例，说明拆分策略与容量评估',
      score: 78,
      strengths: ['量化结果明确', '权衡过程清晰'],
      gaps: ['未提及跨团队协作细节'],
      contradictions: [],
      suggestions: '补充决策失败的备选方案',
    },
    {
      question: '线上事故如何定位与恢复？',
      answer_summary: '按监控→隔离→回滚→复盘四步说明',
      score: 82,
      strengths: ['流程完整', 'RTO 意识强'],
      gaps: [],
      contradictions: ['与简历中“全链路压测”描述略有不一致'],
      suggestions: '统一压测结论口径',
    },
  ],
  communication: { mode: 'voice', note: '语音作答；表达流畅，偶有口头禅' },
  tools: [{ tool: 'code_editor', events: 6, note: '代码实现完整，复杂度分析准确' }],
  training_plan: ['行为类 STAR 练习 ×3', '系统设计表达框架练习', '矛盾点一致性校准'],
  review: { available: true, used: false },
  export_disclaimer: '模拟训练结果，不代表真实企业录用结论',
} as const;

export const mockPractice = {
  practice_id: 'pr-0001',
  question: '请用一个具体案例说明你如何推动跨团队协作',
  variant: '如果对方团队优先级冲突，你会怎么办？',
  hint: '使用 STAR：情境-任务-行动-结果',
  framework: ['情境（S）：时间、背景、干系人', '任务（T）：你的职责与目标', '行动（A）：具体动作与沟通方式', '结果（R）：量化结果与复盘'],
  example: '在订单中心重构中，与算法团队就降级策略达成一致……',
} as const;

export const mockLibrary = {
  resumes: [
    { material_id: 'r-lib-1', name: '合成候选人简历 v1', company: '合成科技', job_title: '后端工程师', version: 3, confirmed_at: '2026-07-28T10:00:00Z' },
    { material_id: 'r-lib-2', name: '英文简历 v1', company: 'Global Sample Inc.', job_title: 'Frontend Engineer', version: 1, confirmed_at: '2026-08-01T01:00:00Z' },
  ],
  jobs: [
    { material_id: 'j-lib-1', name: '后端工程师 JD', company: '合成科技', job_title: '后端工程师', version: 2, confirmed_at: '2026-07-28T10:05:00Z' },
    { material_id: 'j-lib-2', name: '高级产品经理 JD', company: '示例数据公司', job_title: '高级产品经理', version: 1, confirmed_at: '2026-07-30T03:10:00Z' },
  ],
  interviews: [
    { project_id: 'p-0006', name: '全流程训练（已完成）', completed_at: '2026-07-24T10:00:00Z', rounds: 3 },
    { project_id: 'p-0001', name: '后端工程师面试训练', status: IN_SESSION, active_device: '浏览器 · Windows' },
  ],
  training: {
    weaknesses: ['behavioral_collaboration', 'experience_evidence'],
    retry_trail: [
      { round: 1, attempt: 1, status: 'FAIL', score: 52 },
      { round: 1, attempt: 2, status: 'PASS', score: 71 },
    ],
  },
} as const;

export const mockConsents = [
  { consent_type: 'core_service', granted: true, required: true, updated_at: '2026-07-20T02:00:00Z' },
  { consent_type: 'raw_av_recording', granted: false, required: false, updated_at: '2026-07-20T02:00:00Z' },
  { consent_type: 'org_sharing', granted: false, required: false, updated_at: '2026-07-20T02:00:00Z' },
  { consent_type: 'non_essential_analytics', granted: true, required: false, updated_at: '2026-07-20T02:00:00Z' },
  { consent_type: 'model_training', granted: false, required: false, updated_at: '2026-07-20T02:00:00Z' },
  { consent_type: 'marketing', granted: false, required: false, updated_at: '2026-07-20T02:00:00Z' },
] as const;

export const mockPreferences = {
  ui_language: 'zh-CN',
  interview_language: 'zh-CN',
  identity: { email: 'candidate@example.com', providers: ['email'] },
} as const;

export const mockBilling = {
  balance_minutes: 46,
  plan: 'free',
  plans: [
    { id: 'free', name: 'freePlan', minutes: 60, price: 0, currency: 'CNY', per_minute: false },
    { id: 'pack', name: 'projectPack', minutes: 180, price: 39.0, currency: 'CNY', per_minute: false },
    { id: 'pro', name: 'proPlan', minutes: 600, price: 99.0, currency: 'CNY', per_minute: false },
    { id: 'topup', name: 'topup', minutes: 120, price: 29.0, currency: 'CNY', per_minute: false },
  ],
  ledger: [
    { entry_id: 'u-1', entry_type: 'reserve', seconds: 1800, reason: '第 1 轮预留', created_at: '2026-08-01T09:30:00Z' },
    { entry_id: 'u-2', entry_type: 'consume', seconds: 482, reason: '第 1 轮实际使用', created_at: '2026-08-01T09:38:00Z' },
    { entry_id: 'u-3', entry_type: 'release', seconds: 1318, reason: '第 1 轮释放', created_at: '2026-08-01T09:38:00Z' },
  ],
  orders: [
    { order_id: 'o-1', project: '后端工程师面试训练', amount: 39.0, status: 'paid', created_at: '2026-07-28T10:10:00Z' },
  ],
} as const;

export const mockOrg = {
  org_id: 'org-1',
  name: '示例训练机构',
  assignments: [
    { assignment_id: 'a-1', title: '后端岗位模拟面试训练', deadline: '2026-09-01', quota_minutes: 3000, members: 24, my_status: 'in_progress' },
    { assignment_id: 'a-2', title: '产品经理结构化表达训练', deadline: '2026-10-01', quota_minutes: 1500, members: 12, my_status: 'completed_not_shared' },
  ],
  members: [
    { name: '机构管理员', role: 'admin', email: 'admin@example.org', joined_at: '2026-07-01' },
    { name: '指导老师 A', role: 'coach', email: 'coach-a@example.org', joined_at: '2026-07-02' },
    { name: '候选学员', role: 'candidate', email: 'candidate@example.com', joined_at: '2026-07-10' },
  ],
  aggregates: {
    completion_rate: 68,
    dimension_trends: [
      { dimension: 'professional_competence', before: 61, after: 74 },
      { dimension: 'communication', before: 64, after: 71 },
      { dimension: 'problem_solving', before: 59, after: 69 },
    ],
    small_groups_hidden: true,
  },
  audit: [
    { actor: '指导老师 A', action: 'viewed_aggregate', resource: '维度聚合（匿名）', at: '2026-08-01T08:00:00Z' },
    { actor: '机构管理员', action: 'invited_member', resource: '成员邀请', at: '2026-08-01T07:00:00Z' },
  ],
  shares: [
    { assignment_id: 'a-1', scope: '雷达图与维度得分', expires_at: '2026-09-30', status: 'active' },
  ],
} as const;

export const mockAdmin = {
  regions: [
    { region: 'cn', online_rooms: 12, queue: 0, capacity: 96, slo: '99.95%', error_budget: '72%' },
    { region: 'eu', online_rooms: 4, queue: 0, capacity: 64, slo: '99.95%', error_budget: '81%' },
    { region: 'intl', online_rooms: 7, queue: 1, capacity: 80, slo: '99.9%', error_budget: '64%' },
  ],
  providers: [
    { provider_id: 'llm_cn_primary', capability: 'LLM', status: 'open', latency_ms: 420, error_rate: '0.02%' },
    { provider_id: 'asr_cn_primary', capability: 'ASR', status: 'open', latency_ms: 180, error_rate: '0.05%' },
    { provider_id: 'avatar_cn_primary', capability: 'Avatar', status: 'half_open', latency_ms: 320, error_rate: '1.2%' },
  ],
  tickets: [
    { ticket_id: 'tk-1', type: 'fault_refund', status: 'pending_approval', created_at: '2026-08-01T10:00:00Z', amount: 12.0 },
  ],
} as const;
