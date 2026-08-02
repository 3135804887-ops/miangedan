import { render, type RenderResult } from '@testing-library/react';
import { NextIntlClientProvider } from 'next-intl';
import type { ReactElement } from 'react';

import batch1En from '@mgd/i18n/messages/en-US/batch1.json' with { type: 'json' };
import commonEn from '@mgd/i18n/messages/en-US/common.json' with { type: 'json' };
import errorEn from '@mgd/i18n/messages/en-US/error.json' with { type: 'json' };
import batch1Zh from '@mgd/i18n/messages/zh-CN/batch1.json' with { type: 'json' };
import commonZh from '@mgd/i18n/messages/zh-CN/common.json' with { type: 'json' };
import errorZh from '@mgd/i18n/messages/zh-CN/error.json' with { type: 'json' };

export type TestLocale = 'zh-CN' | 'en-US';

const MESSAGES = {
  'zh-CN': { batch1: batch1Zh, common: commonZh, error: errorZh },
  'en-US': { batch1: batch1En, common: commonEn, error: errorEn },
} as const;

export function renderWithIntl(element: ReactElement, locale: TestLocale = 'zh-CN'): RenderResult {
  return render(
    <NextIntlClientProvider locale={locale} messages={MESSAGES[locale]} timeZone="Asia/Shanghai">
      {element}
    </NextIntlClientProvider>,
  );
}
