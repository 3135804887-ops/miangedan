import { http, HttpResponse } from 'msw';
import type { RequestHandler } from 'msw';

import { API_BASE_PATH } from '../../lib/api-fetch.ts';
import { PROJECT_STATUSES, SESSION_STATUSES } from '@mgd/domain-states';
import {
  mockAdmin,
  mockBilling,
  mockConsents,
  mockLibrary,
  mockOrg,
  mockPlan,
  mockPractice,
  mockPreferences,
  mockProjects,
  mockReport,
  mockResult,
  mockSession,
  synthetic,
} from '../data.ts';

/** 合成错误信封，形状与 openapi components.schemas.Error 一致。 */
export function errorEnvelope(code: string, traceId = 'synthetic-trace-0001') {
  return {
    synthetic: true,
    error: {
      code,
      message: '合成数据：用于驱动五态测试，不含任何真实个人信息。',
      trace_id: traceId,
      data_region: 'cn',
    },
  };
}

/** 501 占位：契约已定义但后端服务尚未实现时返回，前端保持占位文案。 */
export function notImplemented(code = 'internal') {
  return HttpResponse.json(errorEnvelope(code), { status: 501 });
}

/** 默认 handler：未被具体页面 handler 覆盖的请求返回 501，避免测试静默通过。 */
export const fallbackHandlers: RequestHandler[] = [
  http.all(`${API_BASE_PATH}/*`, () => notImplemented('internal')),
];

export const handlers: RequestHandler[] = [
  // ---- 身份（SCR-02） ----
  http.post(`${API_BASE_PATH}/v1/identity/email/challenges`, () =>
    HttpResponse.json({ ...synthetic, challenge_id: 'ch-1', resend_after_seconds: 60 }, { status: 201 }),
  ),
  http.post(`${API_BASE_PATH}/v1/identity/email/challenges/:id/verify`, () =>
    HttpResponse.json({
      ...synthetic,
      session_id: 'biz-1',
      access_token: 'synthetic.biz.token',
      refresh_token: 'synthetic.refresh.token',
      expires_in: 3600,
    }),
  ),
  http.post(`${API_BASE_PATH}/v1/identity/oauth/:provider/verify`, () =>
    HttpResponse.json({
      ...synthetic,
      session_id: 'biz-1',
      access_token: 'synthetic.biz.token',
      refresh_token: 'synthetic.refresh.token',
      expires_in: 3600,
    }),
  ),
  http.post(`${API_BASE_PATH}/v1/identity/sessions/refresh`, () =>
    HttpResponse.json({ ...synthetic, access_token: 'synthetic.biz.token.2', expires_in: 3600 }),
  ),
  http.get(`${API_BASE_PATH}/v1/identity/account`, () =>
    HttpResponse.json({ ...synthetic, account: { user_id: 'user-001', email: 'candidate@example.com', data_region: 'cn' } }),
  ),
  http.patch(`${API_BASE_PATH}/v1/identity/account`, () =>
    HttpResponse.json({ ...synthetic, account: { user_id: 'user-001', email: 'candidate@example.com', data_region: 'cn' } }),
  ),

  // ---- 工作台（SCR-03） ----
  http.get(`${API_BASE_PATH}/v1/projects`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockProjects], next_cursor: null }),
  ),
  http.get(`${API_BASE_PATH}/v1/projects/:id`, ({ params }) => {
    const p = mockProjects.find((x) => x.project_id === params.id);
    return p === undefined
      ? HttpResponse.json(errorEnvelope('not_found'), { status: 404 })
      : HttpResponse.json({ ...synthetic, project: p });
  }),
  http.post(`${API_BASE_PATH}/v1/projects`, () =>
    HttpResponse.json({ ...synthetic, project: mockProjects[0] }, { status: 201 }),
  ),
  http.patch(`${API_BASE_PATH}/v1/projects/:id`, () =>
    HttpResponse.json({ ...synthetic, project: mockProjects[0] }),
  ),
  http.post(`${API_BASE_PATH}/v1/projects/:id/duplicate`, () =>
    HttpResponse.json({ ...synthetic, project: mockProjects[1] }, { status: 201 }),
  ),

  // ---- 材料上传与解析（SCR-04/05） ----
  http.post(`${API_BASE_PATH}/v1/uploads/resumes`, () =>
    HttpResponse.json({ ...synthetic, upload_id: '00000000-0000-7000-8000-000000000012', status: 'scanning', data_region: 'cn' }, { status: 201 }),
  ),
  http.get(`${API_BASE_PATH}/v1/uploads/:id`, () =>
    HttpResponse.json({ ...synthetic, upload_id: '00000000-0000-7000-8000-000000000012', status: 'accepted', data_region: 'cn' }),
  ),
  http.post(`${API_BASE_PATH}/v1/uploads/:id:retry`, () =>
    HttpResponse.json({ ...synthetic, upload_id: '00000000-0000-7000-8000-000000000012', status: 'accepted' }),
  ),
  http.post(`${API_BASE_PATH}/v1/parsing/resumes`, () =>
    HttpResponse.json({ ...synthetic, resume_id: '00000000-0000-7000-8000-000000000013', version: 1, status: 'awaiting_confirmation' }, { status: 201 }),
  ),
  http.get(`${API_BASE_PATH}/v1/parsing/resumes/:resumeId/versions/:version`, () =>
    HttpResponse.json({ ...synthetic, resume_id: '00000000-0000-7000-8000-000000000013', version: 1, confirmed_by_user: false, profile: {} }),
  ),
  http.post(`${API_BASE_PATH}/v1/resumes`, () =>
    HttpResponse.json({ ...synthetic, resume_id: '00000000-0000-7000-8000-000000000013', version: 1 }, { status: 201 }),
  ),
  http.get(`${API_BASE_PATH}/v1/resumes/:resumeId/versions/:version`, () =>
    HttpResponse.json({ ...synthetic, resume_id: '00000000-0000-7000-8000-000000000013', version: 1, confirmed_by_user: false, profile: {} }),
  ),
  http.post(`${API_BASE_PATH}/v1/resumes/:resumeId/versions/:version/confirm`, () =>
    HttpResponse.json({ ...synthetic, resume_id: '00000000-0000-7000-8000-000000000013', version: 1, confirmed_by_user: true }),
  ),
  http.post(`${API_BASE_PATH}/v1/jobs`, () =>
    HttpResponse.json({ ...synthetic, job_id: '00000000-0000-7000-8000-000000000014', version: 1 }, { status: 201 }),
  ),
  http.get(`${API_BASE_PATH}/v1/jobs/:jobId/versions/:version`, () =>
    HttpResponse.json({ ...synthetic, job_id: '00000000-0000-7000-8000-000000000014', version: 1, confirmed_by_user: false, profile: {} }),
  ),
  http.post(`${API_BASE_PATH}/v1/jobs/:jobId/versions/:version/confirm`, () =>
    HttpResponse.json({ ...synthetic, job_id: '00000000-0000-7000-8000-000000000014', version: 1, confirmed_by_user: true }),
  ),
  http.post(`${API_BASE_PATH}/v1/jobs/material-readiness:assess`, () =>
    HttpResponse.json({ ...synthetic, degraded_mode: 'full', impacts: [], requires_consent: false }),
  ),
  http.post(`${API_BASE_PATH}/v1/jobs/degraded-consents`, () =>
    HttpResponse.json({ ...synthetic, consent_id: 'consent-0001', accepted: true }, { status: 201 }),
  ),
  http.post(`${API_BASE_PATH}/v1/jobs:infer-from-resume`, () =>
    HttpResponse.json({ ...synthetic, job_id: '00000000-0000-7000-8000-000000000014', version: 1, ai_derived: true }, { status: 201 }),
  ),

  // ---- 公开流程来源 ----
  http.post(`${API_BASE_PATH}/v1/sources/search`, () =>
    HttpResponse.json({ ...synthetic, items: [], flow_uses_generic_template: true, ai_derived: true }),
  ),
  http.get(`${API_BASE_PATH}/v1/sources`, () =>
    HttpResponse.json({ ...synthetic, items: [], next_cursor: null }),
  ),

  // ---- 计划（SCR-06） ----
  http.post(`${API_BASE_PATH}/v1/projects/:id/plan:generate`, () =>
    HttpResponse.json({ ...synthetic, plan: { ...mockPlan, project_id: 'p-0003', frozen: false } }, { status: 201 }),
  ),
  http.get(`${API_BASE_PATH}/v1/projects/:id/plan`, () =>
    HttpResponse.json({ ...synthetic, plan: mockPlan }),
  ),
  http.patch(`${API_BASE_PATH}/v1/projects/:id/plan`, () =>
    HttpResponse.json({ ...synthetic, plan: mockPlan }),
  ),
  http.post(`${API_BASE_PATH}/v1/projects/:id/plan:confirm`, () =>
    HttpResponse.json({ ...synthetic, project: { ...mockProjects[2], status: PROJECT_STATUSES[7] } }),
  ),

  // ---- 会话与房间（SCR-07/08/09） ----
  http.post(`${API_BASE_PATH}/v1/projects/:id/rounds/:seq/session`, () =>
    HttpResponse.json(
      { ...synthetic, session_id: mockSession.session_id, room_url: 'wss://stub.example/room', room_token: 'synthetic.media.token', room_token_expires_at: '2026-08-02T23:30:00Z', data_region: 'cn' },
      { status: 201 },
    ),
  ),
  http.get(`${API_BASE_PATH}/v1/sessions/:id`, () =>
    HttpResponse.json({ ...synthetic, session: mockSession }),
  ),
  http.get(`${API_BASE_PATH}/v1/sessions/:id/precheck`, () =>
    HttpResponse.json({ ...synthetic, precheck: { frozen: false, input_modes: ['voice', 'text'], accommodations: [], device_report: { camera_ok: true, mic_ok: true, network_rated: 'good' } } }),
  ),
  http.post(`${API_BASE_PATH}/v1/sessions/:id/precheck/freeze`, () =>
    HttpResponse.json({ ...synthetic, session_id: mockSession.session_id, frozen: true, input_modes: ['voice', 'text'], accommodations: [], device_report: { camera_ok: true, mic_ok: true, network_rated: 'good' } }),
  ),
  http.post(`${API_BASE_PATH}/v1/sessions/:id/timer/pause`, () =>
    HttpResponse.json({ ...synthetic, session: { ...mockSession, room_status: SESSION_STATUSES[4] } }),
  ),
  http.post(`${API_BASE_PATH}/v1/sessions/:id/timer/resume`, () =>
    HttpResponse.json({ ...synthetic, session: mockSession }),
  ),
  http.post(`${API_BASE_PATH}/v1/sessions/:id/downgrade/offer`, () =>
    HttpResponse.json({ ...synthetic, prompt_id: 'dg-1' }),
  ),
  http.post(`${API_BASE_PATH}/v1/sessions/:id/downgrade/accept`, () =>
    HttpResponse.json({ ...synthetic, session: { ...mockSession, room_status: SESSION_STATUSES[7] } }),
  ),
  http.post(`${API_BASE_PATH}/v1/sessions/:id/downgrade/decline`, () =>
    HttpResponse.json({ ...synthetic, session: { ...mockSession, room_status: SESSION_STATUSES[9] } }),
  ),
  http.post(`${API_BASE_PATH}/v1/sessions/:id/end`, () =>
    HttpResponse.json({ ...synthetic, session: { ...mockSession, room_status: SESSION_STATUSES[9] } }),
  ),
  http.post(`${API_BASE_PATH}/v1/sessions/:id/reconnect`, () =>
    HttpResponse.json({ ...synthetic, session: mockSession }),
  ),
  http.get(`${API_BASE_PATH}/v1/sessions/:id/transcripts`, () =>
    HttpResponse.json({ ...synthetic, items: [], next_cursor: null }),
  ),
  http.post(`${API_BASE_PATH}/v1/sessions/:id/transcripts`, () =>
    HttpResponse.json({ ...synthetic, item: { transcript_id: 'tr-1', turn_index: 1, role: 'candidate', text: '合成转写' } }, { status: 201 }),
  ),
  http.post(`${API_BASE_PATH}/v1/sessions/:id/revisions`, () =>
    HttpResponse.json({ ...synthetic, revision: { revision_id: 'rv-1', turn_index: 1, revised_text: '合成修订文本', window_closed: false } }, { status: 201 }),
  ),
  http.get(`${API_BASE_PATH}/v1/sessions/:id/turns/:turnIndex`, () =>
    HttpResponse.json({ ...synthetic, turn: { turn_index: 1, question_played: true, answer_submitted: false } }),
  ),
  http.post(`${API_BASE_PATH}/v1/sessions/:id/turns/:turnIndex/freeze`, () =>
    HttpResponse.json({ ...synthetic, turn: { turn_index: 1, frozen: true } }),
  ),
  http.get(`${API_BASE_PATH}/v1/sessions/:id/tools`, () =>
    HttpResponse.json({ ...synthetic, items: [{ tool_key: 'code_editor', configured: true, active: false }] }),
  ),
  http.post(`${API_BASE_PATH}/v1/sessions/:id/tools/:toolKey/activate`, () =>
    HttpResponse.json({ ...synthetic, tool: { tool_key: 'code_editor', configured: true, active: true } }),
  ),
  http.post(`${API_BASE_PATH}/v1/sessions/:id/tools/:toolKey/events`, () =>
    HttpResponse.json({ ...synthetic, event: { event_id: 'tool-ev-1', tool_key: 'code_editor', kind: 'edit' } }, { status: 201 }),
  ),
  http.post(`${API_BASE_PATH}/v1/sessions/:id/device-transfer`, () =>
    HttpResponse.json({ ...synthetic, session: mockSession }),
  ),

  // ---- 结果、评分与复核（SCR-10） ----
  http.get(`${API_BASE_PATH}/v1/projects/:id/rounds/:seq/result`, () =>
    HttpResponse.json({ ...synthetic, result: mockResult }),
  ),
  http.get(`${API_BASE_PATH}/v1/projects/:id/rounds/:seq/scores`, () =>
    HttpResponse.json({ ...synthetic, items: [], next_cursor: null }),
  ),
  http.post(`${API_BASE_PATH}/v1/projects/:id/rounds/:seq/review`, () =>
    HttpResponse.json({ ...synthetic, review: { review_id: 'rvw-1', old_total: 58, new_total: 66, reasons: ['证据补充后重新计算'] } }, { status: 201 }),
  ),
  http.post(`${API_BASE_PATH}/v1/projects/:id/rounds/:seq/retry`, () =>
    HttpResponse.json({ ...synthetic, attempt: { attempt_id: 'retry-1', status: 'retry_scheduled', locked_dimensions: [] } }, { status: 201 }),
  ),

  // ---- 报告（SCR-11） ----
  http.get(`${API_BASE_PATH}/v1/projects/:id/report`, () =>
    HttpResponse.json({ ...synthetic, report: mockReport }),
  ),
  http.post(`${API_BASE_PATH}/v1/projects/:id/report:regenerate`, () =>
    HttpResponse.json({ ...synthetic, report: mockReport }),
  ),
  http.post(`${API_BASE_PATH}/v1/projects/:id/report/export`, () =>
    HttpResponse.json({ ...synthetic, task_id: 'exp-report-1', status: 'queued' }, { status: 202 }),
  ),

  // ---- 练习（SCR-12） ----
  http.post(`${API_BASE_PATH}/v1/projects/:id/practice`, () =>
    HttpResponse.json({ ...synthetic, practice: mockPractice }, { status: 201 }),
  ),
  http.post(`${API_BASE_PATH}/v1/practice/:id/end`, () =>
    HttpResponse.json({ ...synthetic, practice: { ...mockPractice, ended_at: '2026-08-02T23:00:00Z' } }),
  ),

  // ---- 资产（SCR-13） ----
  http.get(`${API_BASE_PATH}/v1/library/resumes`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockLibrary.resumes] }),
  ),
  http.delete(`${API_BASE_PATH}/v1/library/resumes/:id`, () =>
    HttpResponse.json({ ...synthetic, deleted: true }, { status: 202 }),
  ),
  http.get(`${API_BASE_PATH}/v1/library/jobs`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockLibrary.jobs] }),
  ),
  http.delete(`${API_BASE_PATH}/v1/library/jobs/:id`, () =>
    HttpResponse.json({ ...synthetic, deleted: true }, { status: 202 }),
  ),

  // ---- 设置（SCR-14） ----
  http.get(`${API_BASE_PATH}/v1/me/preferences`, () =>
    HttpResponse.json({ ...synthetic, preferences: mockPreferences }),
  ),
  http.put(`${API_BASE_PATH}/v1/me/preferences`, () =>
    HttpResponse.json({ ...synthetic, preferences: mockPreferences }),
  ),
  http.get(`${API_BASE_PATH}/v1/consent/grants`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockConsents] }),
  ),
  http.put(`${API_BASE_PATH}/v1/consent/grants/:type`, () =>
    HttpResponse.json({ ...synthetic, granted: true, consent_type: 'core_service', version: 2 }, { status: 201 }),
  ),
  http.post(`${API_BASE_PATH}/v1/consent/grants/:type/withdrawals`, () =>
    HttpResponse.json({ ...synthetic, granted: false, consent_type: 'org_sharing', version: 3 }),
  ),
  http.get(`${API_BASE_PATH}/v1/consent/grants/:type/history`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockConsents] }),
  ),
  http.get(`${API_BASE_PATH}/v1/me/export`, () =>
    HttpResponse.json({ ...synthetic, task_id: 'exp-1', status: 'queued' }, { status: 202 }),
  ),
  http.post(`${API_BASE_PATH}/v1/me/deletion`, () =>
    HttpResponse.json({ ...synthetic, task_id: 'del-1', status: 'queued' }, { status: 202 }),
  ),
  http.get(`${API_BASE_PATH}/v1/deletion-tasks/:taskId`, () =>
    HttpResponse.json({ ...synthetic, task_id: 'del-1', status: 'in_progress', progress: [{ layer: 'database', status: 'done' }] }),
  ),

  // ---- 购买（SCR-15） ----
  http.get(`${API_BASE_PATH}/v1/entitlements`, () =>
    HttpResponse.json({ ...synthetic, entitlement: { balance_minutes: mockBilling.balance_minutes, plan: mockBilling.plan } }),
  ),
  http.post(`${API_BASE_PATH}/v1/quotes`, () =>
    HttpResponse.json({ ...synthetic, quote: { quote_id: 'q-1', status: 'PRESENTED', total: mockBilling.orders[0]?.amount ?? 0 } }, { status: 201 }),
  ),
  http.post(`${API_BASE_PATH}/v1/orders`, () =>
    HttpResponse.json({ ...synthetic, order: mockBilling.orders[0] }, { status: 201 }),
  ),
  http.get(`${API_BASE_PATH}/v1/orders/:orderId`, () =>
    HttpResponse.json({ ...synthetic, order: mockBilling.orders[0] }),
  ),
  http.get(`${API_BASE_PATH}/v1/pricing/:region`, () =>
    HttpResponse.json({ ...synthetic, pricing: { region: 'cn', currency: 'CNY', plans: [...mockBilling.plans] } }),
  ),
  http.get(`${API_BASE_PATH}/v1/usage-ledger`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockBilling.ledger], next_cursor: null }),
  ),
  http.get(`${API_BASE_PATH}/v1/subscription`, () =>
    HttpResponse.json({ ...synthetic, subscription: { plan: 'free', status: 'active', auto_renew: false, balance_minutes: mockBilling.balance_minutes } }),
  ),
  http.post(`${API_BASE_PATH}/v1/subscription/cancel`, () =>
    HttpResponse.json({ ...synthetic, subscription: { plan: 'free', status: 'cancelled', auto_renew: false } }),
  ),
  http.put(`${API_BASE_PATH}/v1/subscription/auto-renew`, () =>
    HttpResponse.json({ ...synthetic, subscription: { plan: 'pro', status: 'active', auto_renew: true } }),
  ),

  // ---- 机构（SCR-16） ----
  http.get(`${API_BASE_PATH}/v1/orgs/:orgId`, () =>
    HttpResponse.json({ ...synthetic, org: { org_id: 'org-0001', name: '合成科技学院', status: 'active' } }),
  ),
  http.get(`${API_BASE_PATH}/v1/orgs/:orgId/members`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockOrg.members] }),
  ),
  http.post(`${API_BASE_PATH}/v1/orgs/:orgId/members/invite`, () =>
    HttpResponse.json({ ...synthetic, invitation_id: 'inv-1', status: 'pending' }, { status: 201 }),
  ),
  http.delete(`${API_BASE_PATH}/v1/orgs/:orgId/members/:userId`, () =>
    HttpResponse.json({ ...synthetic, removed: true }),
  ),
  http.post(`${API_BASE_PATH}/v1/orgs/:orgId/members/:userId/role`, () =>
    HttpResponse.json({ ...synthetic, user_id: 'user-002', role: 'instructor' }),
  ),
  http.post(`${API_BASE_PATH}/v1/orgs/:orgId/assignments`, () =>
    HttpResponse.json({ ...synthetic, assignment: mockOrg.assignments[0] }, { status: 201 }),
  ),
  http.get(`${API_BASE_PATH}/v1/orgs/:orgId/assignments/:assignmentId`, () =>
    HttpResponse.json({ ...synthetic, assignment: mockOrg.assignments[0] }),
  ),
  http.post(`${API_BASE_PATH}/v1/orgs/:orgId/assignments/:assignmentId/publish`, () =>
    HttpResponse.json({ ...synthetic, assignment: { ...mockOrg.assignments[0], status: 'published' } }),
  ),
  http.post(`${API_BASE_PATH}/v1/orgs/:orgId/assignments/:assignmentId/close`, () =>
    HttpResponse.json({ ...synthetic, assignment: { ...mockOrg.assignments[0], status: 'closed' } }),
  ),
  http.get(`${API_BASE_PATH}/v1/orgs/:orgId/aggregates`, () =>
    HttpResponse.json({ ...synthetic, aggregates: mockOrg.aggregates }),
  ),
  http.get(`${API_BASE_PATH}/v1/orgs/:orgId/audits`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockOrg.audit] }),
  ),
  http.post(`${API_BASE_PATH}/v1/orgs/:orgId/leave`, () =>
    HttpResponse.json({ ...synthetic, left: true, records_kept: true }),
  ),
  http.post(`${API_BASE_PATH}/v1/orgs/:orgId/status`, () =>
    HttpResponse.json({ ...synthetic, org: { org_id: 'org-0001', name: '合成科技学院', status: 'suspended' } }),
  ),
  http.post(`${API_BASE_PATH}/v1/assignments/:assignmentId/shares`, () =>
    HttpResponse.json({ ...synthetic, share: { share_id: 'share-1', scope: 'radar', expires_at: '2026-09-01T00:00:00Z' } }, { status: 201 }),
  ),
  http.get(`${API_BASE_PATH}/v1/assignments/:assignmentId/shares`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockOrg.shares] }),
  ),
  http.delete(`${API_BASE_PATH}/v1/assignments/:assignmentId/shares/:shareId`, () =>
    HttpResponse.json({ ...synthetic, revoked: true }),
  ),
  http.post(`${API_BASE_PATH}/v1/orgs/invitations/:invitationId/accept`, () =>
    HttpResponse.json({ ...synthetic, org_id: 'org-0001', role: 'instructor' }),
  ),

  // ---- 后台（SCR-17） ----
  http.get(`${API_BASE_PATH}/v1/admin/regions`, () =>
    HttpResponse.json({ ...synthetic, regions: [...mockAdmin.regions] }),
  ),
  http.get(`${API_BASE_PATH}/v1/admin/regions/:region/rooms`, () =>
    HttpResponse.json({ ...synthetic, rooms: [] }),
  ),
  http.get(`${API_BASE_PATH}/v1/admin/providers`, () =>
    HttpResponse.json({ ...synthetic, providers: [...mockAdmin.providers] }),
  ),
  http.put(`${API_BASE_PATH}/v1/admin/providers/:providerId/status`, () =>
    HttpResponse.json({ ...synthetic, provider: { provider_id: 'llm_cn_primary', status: 'disabled' } }),
  ),
  http.get(`${API_BASE_PATH}/v1/admin/audit-logs`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockOrg.audit] }),
  ),
  http.get(`${API_BASE_PATH}/v1/admin/data-rights`, () =>
    HttpResponse.json({ ...synthetic, items: [] }),
  ),
  http.post(`${API_BASE_PATH}/v1/admin/break-glass`, () =>
    HttpResponse.json({ ...synthetic, glass_id: 'glass-1', status: 'open', expires_at: '2026-08-03T18:00:00Z' }, { status: 201 }),
  ),
  http.post(`${API_BASE_PATH}/v1/admin/break-glass/:glassId/review`, () =>
    HttpResponse.json({ ...synthetic, glass_id: 'glass-1', status: 'reviewed' }),
  ),
  http.post(`${API_BASE_PATH}/v1/admin/tickets`, () =>
    HttpResponse.json({ ...synthetic, ticket_id: 'ticket-1', status: 'open' }, { status: 201 }),
  ),
  http.get(`${API_BASE_PATH}/v1/admin/tickets/:ticketId`, () =>
    HttpResponse.json({ ...synthetic, ticket: { ticket_id: 'ticket-1', status: 'open', transcript_authorized: false } }),
  ),
  http.post(`${API_BASE_PATH}/v1/admin/tickets/:ticketId/transcript-auth`, () =>
    HttpResponse.json({ ...synthetic, ticket_id: 'ticket-1', transcript_authorized: true, expires_at: '2026-08-04T00:00:00Z' }),
  ),
];
