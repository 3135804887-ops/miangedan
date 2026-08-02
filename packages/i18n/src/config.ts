/**
 * 语言配置（需求 G3）。
 * 界面语言由 URL 前缀决定；面试语言是独立字段（SCR-14 分别设置），两者不可互相赋值。
 */

/** 与 docs/api/openapi.yaml 的 components.schemas.Language 一致。 */
export const SUPPORTED_LOCALES = ['zh-CN', 'en-US'] as const;

export type Locale = (typeof SUPPORTED_LOCALES)[number];

/** 默认界面语言（首发主语言）。 */
export const DEFAULT_LOCALE: Locale = 'zh-CN';

/** 最终回退语言（DESIGN-SYSTEM 第 11 节：未知语言回退 en-US）。 */
export const FALLBACK_LOCALE: Locale = 'en-US';

const LOCALE_SET: ReadonlySet<string> = new Set(SUPPORTED_LOCALES);

export function isLocale(value: string): value is Locale {
  return LOCALE_SET.has(value);
}

/** 解析路径首段为界面语言；不支持的取值回退到 en-US（G3 第 3 条）。 */
export function resolveLocale(candidate: string | undefined): Locale {
  if (candidate !== undefined && isLocale(candidate)) return candidate;
  return FALLBACK_LOCALE;
}

/**
 * 面试语言：与界面语言取值域相同但语义独立，使用品牌类型防止误用互相赋值。
 * 用法：asInterviewLanguage(project.interview_language)
 */
declare const INTERVIEW_LANGUAGE_BRAND: unique symbol;

export type InterviewLanguage = Locale & { readonly [INTERVIEW_LANGUAGE_BRAND]: true };

export function asInterviewLanguage(value: Locale): InterviewLanguage {
  return value as InterviewLanguage;
}

/** i18n 命名空间：按页面组切分，降低批次间合并冲突（design.md 第 7 节）。 */
export const MESSAGE_NAMESPACES = [
  'common',
  'error',
  'scr01-landing',
  'scr02-auth',
  'scr03-dashboard',
  'scr04-project-new',
  'scr05-review',
  'scr06-plan',
  'scr07-precheck',
  'scr08-room',
  'scr09-overlay',
  'scr10-result',
  'scr11-report',
  'scr12-practice',
  'scr13-library',
  'scr14-settings',
  'scr15-billing',
  'scr16-org',
  'scr17-admin',
] as const;

export type MessageNamespace = (typeof MESSAGE_NAMESPACES)[number];
