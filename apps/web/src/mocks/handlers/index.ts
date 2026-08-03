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

/** 默认 handler：未被具体页面 handler 覆盖的请求返回 501，避免测试静默通过。 */
export const fallbackHandlers: RequestHandler[] = [
  http.all(`${API_BASE_PATH}/*`, () =>
    HttpResponse.json(errorEnvelope('internal', 'synthetic-unhandled'), { status: 501 }),
  ),
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

  // ---- 计划（SCR-06） ----
  http.post(`${API_BASE_PATH}/v1/projects/:id/plan:generate`, () =>
    HttpResponse.json({ ...synthetic, plan: { ...mockPlan, project_id: 'p-0003', frozen: false } }, { status: 201 }),
  ),
  http.get(`${API_BASE_PATH}/v1/projects/:id/plan`, () =>
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

  // ---- 结果与报告（SCR-10/11） ----
  http.get(`${API_BASE_PATH}/v1/projects/:id/rounds/:seq/result`, () =>
    HttpResponse.json({ ...synthetic, result: mockResult }),
  ),
  http.get(`${API_BASE_PATH}/v1/projects/:id/report`, () =>
    HttpResponse.json({ ...synthetic, report: mockReport }),
  ),
  http.post(`${API_BASE_PATH}/v1/projects/:id/report:regenerate`, () =>
    HttpResponse.json({ ...synthetic, report: mockReport }),
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
  http.get(`${API_BASE_PATH}/v1/library/jobs`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockLibrary.jobs] }),
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
  http.post(`${API_BASE_PATH}/v1/consent/grants/:type/withdrawals`, () =>
    HttpResponse.json({ ...synthetic, granted: false, consent_type: 'org_sharing' }),
  ),
  http.post(`${API_BASE_PATH}/v1/me/export`, () =>
    HttpResponse.json({ ...synthetic, task_id: 'exp-1', status: 'queued' }, { status: 202 }),
  ),
  http.post(`${API_BASE_PATH}/v1/me/deletion`, () =>
    HttpResponse.json({ ...synthetic, task_id: 'del-1', status: 'queued' }, { status: 202 }),
  ),

  // ---- 购买（SCR-15） ----
  http.get(`${API_BASE_PATH}/v1/entitlements`, () =>
    HttpResponse.json({ ...synthetic, entitlement: { balance_minutes: mockBilling.balance_minutes, plan: mockBilling.plan } }),
  ),
  http.get(`${API_BASE_PATH}/v1/quotes`, () =>
    HttpResponse.json({ ...synthetic, plans: [...mockBilling.plans] }),
  ),
  http.get(`${API_BASE_PATH}/v1/usage-ledger`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockBilling.ledger], next_cursor: null }),
  ),
  http.get(`${API_BASE_PATH}/v1/orders`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockBilling.orders] }),
  ),

  // ---- 机构（SCR-16） ----
  http.get(`${API_BASE_PATH}/v1/orgs/:id/assignments`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockOrg.assignments] }),
  ),
  http.get(`${API_BASE_PATH}/v1/orgs/:id/members`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockOrg.members] }),
  ),
  http.get(`${API_BASE_PATH}/v1/orgs/:id/aggregates`, () =>
    HttpResponse.json({ ...synthetic, aggregates: mockOrg.aggregates }),
  ),
  http.get(`${API_BASE_PATH}/v1/orgs/:id/audit-logs`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockOrg.audit] }),
  ),
  http.get(`${API_BASE_PATH}/v1/orgs/:id/shares`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockOrg.shares] }),
  ),

  // ---- 后台（SCR-17） ----
  http.get(`${API_BASE_PATH}/v1/admin/monitor`, () =>
    HttpResponse.json({ ...synthetic, regions: [...mockAdmin.regions] }),
  ),
  http.get(`${API_BASE_PATH}/v1/admin/providers`, () =>
    HttpResponse.json({ ...synthetic, providers: [...mockAdmin.providers] }),
  ),
  http.get(`${API_BASE_PATH}/v1/admin/tickets`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockAdmin.tickets] }),
  ),
  http.get(`${API_BASE_PATH}/v1/admin/audit-logs`, () =>
    HttpResponse.json({ ...synthetic, items: [...mockOrg.audit] }),
  ),
];
