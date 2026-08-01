/**
 * 路由级加载视图（需求 B0-2 第 3 条）：骨架 + 可访问忙碌标记。
 */

import { Skeleton } from '@mgd/ui';
import { getTranslations } from 'next-intl/server';
import type { ReactNode } from 'react';

export default async function Loading(): Promise<ReactNode> {
  const t = await getTranslations('common');

  return (
    <div data-mgd-view="loading">
      <Skeleton busyLabel={t('state.loadingLabel')} />
    </div>
  );
}
