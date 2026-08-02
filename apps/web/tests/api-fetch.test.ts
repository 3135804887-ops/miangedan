import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';

import { apiFetch } from '../src/lib/api-fetch.ts';
import { mockServer } from '../src/mocks/server.ts';

describe('apiFetch 写操作幂等约束', () => {
  it('把同一用户动作的幂等键原样发送，并解析合成成功响应', async () => {
    const receivedKeys: string[] = [];
    mockServer.use(
      http.post('/api/v1/identity/email/challenges', ({ request }) => {
        receivedKeys.push(request.headers.get('Idempotency-Key') ?? '');
        return HttpResponse.json({ synthetic: true, challenge_id: 'synthetic-challenge' });
      }),
    );

    const idempotencyKey = 'synthetic-action-0001';
    const first = await apiFetch<{ readonly challenge_id: string }>(
      '/v1/identity/email/challenges',
      { method: 'post', idempotencyKey, body: { email: 'synthetic@example.invalid' } },
    );
    const retry = await apiFetch<{ readonly challenge_id: string }>(
      '/v1/identity/email/challenges',
      { method: 'post', idempotencyKey, body: { email: 'synthetic@example.invalid' } },
    );

    expect(receivedKeys).toEqual([idempotencyKey, idempotencyKey]);
    expect(first.ok && first.data.challenge_id).toBe('synthetic-challenge');
    expect(retry.ok && retry.data.challenge_id).toBe('synthetic-challenge');
  });
});
