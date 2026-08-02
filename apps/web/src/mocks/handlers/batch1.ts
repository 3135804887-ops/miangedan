import { PROJECT_STATUSES } from '@mgd/domain-states';
import { http, HttpResponse, type RequestHandler } from 'msw';

import { API_BASE_PATH } from '../../lib/api-fetch.ts';

export type Batch1ProjectScenario = 'ok' | 'empty' | 'failure';

/** SCR-03 列表三场景工厂；所有内容均为 synthetic，字段形状遵循 ProjectPage 契约。 */
export function projectListHandlers(scenario: Batch1ProjectScenario): ReadonlyArray<RequestHandler> {
  return [
    http.get(`${API_BASE_PATH}/projects`, () => {
      if (scenario === 'failure') {
        return HttpResponse.json({
          synthetic: true,
          error: {
            code: 'internal',
            message: 'Synthetic dashboard failure; no personal data is involved.',
            trace_id: 'synthetic-dashboard-failure',
            data_region: 'cn',
          },
        }, { status: 503 });
      }
      return HttpResponse.json({
        synthetic: true,
        data_region: 'cn',
        items: scenario === 'empty' ? [] : [
          {
            project_id: '00000000-0000-4000-8000-000000000101',
            data_region: 'cn',
            name: 'Synthetic product operations interview',
            interview_language: 'zh-CN',
            degraded_mode: 'full',
            status: PROJECT_STATUSES[5],
            current_round_sequence: 2,
            plan_version: 1,
            active_device_id: null,
            assignment_id: null,
            created_at: '2026-07-10T08:00:00Z',
          },
        ],
        next_cursor: null,
      });
    }),
  ];
}

export const batch1Handlers: ReadonlyArray<RequestHandler> = projectListHandlers('ok');
