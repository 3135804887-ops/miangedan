/**
 * 本地化格式化（需求 G3 第 4 条、DESIGN-SYSTEM 第 9 节第 4 条）。
 * 货币金额以最小货币单位传入（对齐 openapi components.schemas.Money.amount 为分/cent）。
 */

import type { Locale } from './config.ts';

export interface Money {
  /** 最小货币单位（分 / cent） */
  readonly amount: number;
  /** ISO 4217，如 CNY、USD */
  readonly currency: string;
}

/** 最小货币单位到主单位的换算基数。零小数货币在此登记。 */
const ZERO_DECIMAL_CURRENCIES: ReadonlySet<string> = new Set(['JPY', 'KRW']);

function minorUnitDivisor(currency: string): number {
  return ZERO_DECIMAL_CURRENCIES.has(currency.toUpperCase()) ? 1 : 100;
}

export function formatMoney(money: Money, locale: Locale): string {
  const divisor = minorUnitDivisor(money.currency);
  const fractionDigits = divisor === 1 ? 0 : 2;

  return new Intl.NumberFormat(locale, {
    style: 'currency',
    currency: money.currency,
    minimumFractionDigits: fractionDigits,
    maximumFractionDigits: fractionDigits,
  }).format(money.amount / divisor);
}

export function formatDate(isoDate: string, locale: Locale): string {
  return new Intl.DateTimeFormat(locale, { dateStyle: 'medium' }).format(new Date(isoDate));
}

export function formatDateTime(isoDate: string, locale: Locale): string {
  return new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }).format(
    new Date(isoDate),
  );
}

/** 秒数展示（额度流水与房间计时用）。 */
export function formatSeconds(totalSeconds: number, locale: Locale): string {
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  const nf = new Intl.NumberFormat(locale, { minimumIntegerDigits: 2 });
  return `${new Intl.NumberFormat(locale).format(minutes)}:${nf.format(seconds)}`;
}
