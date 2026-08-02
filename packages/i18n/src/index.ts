/**
 * @mgd/i18n 公共入口（需求 G3）。
 */

export {
  asInterviewLanguage,
  DEFAULT_LOCALE,
  FALLBACK_LOCALE,
  isLocale,
  MESSAGE_NAMESPACES,
  resolveLocale,
  SUPPORTED_LOCALES,
  type InterviewLanguage,
  type Locale,
  type MessageNamespace,
} from './config.ts';

export {
  formatDate,
  formatDateTime,
  formatMoney,
  formatSeconds,
  type Money,
} from './format.ts';

export {
  LOADING_EXPECTATIONS,
  loadingExpectation,
  type LoadingExpectation,
  type NfrId,
} from './nfr-expectations.ts';
