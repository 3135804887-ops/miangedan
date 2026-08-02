/**
 * 消息资源装载。命名空间按页面组切分（design.md 第 7 节），
 * 批次 0 载入 common 与 error，后续批次逐页增补。
 */

import type { Locale } from '@mgd/i18n';

import commonEn from '@mgd/i18n/messages/en-US/common.json' with { type: 'json' };
import batch1En from '@mgd/i18n/messages/en-US/batch1.json' with { type: 'json' };
import errorEn from '@mgd/i18n/messages/en-US/error.json' with { type: 'json' };
import commonZh from '@mgd/i18n/messages/zh-CN/common.json' with { type: 'json' };
import batch1Zh from '@mgd/i18n/messages/zh-CN/batch1.json' with { type: 'json' };
import errorZh from '@mgd/i18n/messages/zh-CN/error.json' with { type: 'json' };

type MessageTree = Record<string, unknown>;

const BUNDLES: Readonly<Record<Locale, MessageTree>> = {
  'zh-CN': { common: commonZh, error: errorZh, batch1: batch1Zh },
  'en-US': { common: commonEn, error: errorEn, batch1: batch1En },
};

export function loadMessages(locale: Locale): Promise<MessageTree> {
  return Promise.resolve(BUNDLES[locale]);
}
