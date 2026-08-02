#!/usr/bin/env node
/**
 * 翻译键门禁（需求 G3 第 5、6、8 条）。
 *
 * 1. 两语言键集合求对称差，非空即失败（避免运行期回落到原始键名）
 * 2. ICU 占位符集合在两语言必须一致
 * 3. 源码中 t('…') 使用的键必须在两语言均存在
 * 4. 同键值在两语言完全相同时提示需要人工撰写英文文案（白名单除外）
 *
 * 退出码非 0 表示失败。
 */

import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';

const I18N_ROOT = join(import.meta.dirname, '..');
const REPO_ROOT = join(I18N_ROOT, '..', '..');
const MESSAGES_DIR = join(I18N_ROOT, 'messages');
const LOCALES = ['zh-CN', 'en-US'];
const REFERENCE_LOCALE = 'zh-CN';

/** 两语言允许取值相同的键（品牌名、代码类值、纯数字或标点）。 */
const SAME_VALUE_ALLOWED = new Set(['common.brand.name']);

const failures = [];

function fail(message) {
  failures.push(message);
}

/** 以 . 连接的扁平键；忽略以 _ 开头的说明性键（如 _note）。 */
function flatten(value, prefix, sink) {
  for (const [key, child] of Object.entries(value)) {
    if (key.startsWith('_')) continue;
    const path = prefix === '' ? key : `${prefix}.${key}`;
    if (child !== null && typeof child === 'object' && !Array.isArray(child)) {
      flatten(child, path, sink);
    } else if (typeof child === 'string') {
      sink.set(path, child);
    } else {
      fail(`[翻译值类型非法] ${path} 应为字符串，实际为 ${typeof child}`);
    }
  }
}

function loadLocale(locale) {
  const dir = join(MESSAGES_DIR, locale);
  const flat = new Map();
  for (const fileName of readdirSync(dir).sort()) {
    if (!fileName.endsWith('.json')) continue;
    const namespace = fileName.replace(/\.json$/, '');
    const parsed = JSON.parse(readFileSync(join(dir, fileName), 'utf8'));
    flat.set(namespace, new Map());
    const sink = new Map();
    flatten(parsed, '', sink);
    for (const [key, value] of sink) {
      flat.get(namespace).set(key, value);
    }
  }
  return flat;
}

/** 提取 ICU 占位符名：{name}、{name, plural, …}、{name, number, ::currency/CNY} */
function placeholders(message) {
  const names = new Set();
  for (const match of message.matchAll(/\{\s*([A-Za-z0-9_]+)\s*[,}]/g)) {
    names.add(match[1]);
  }
  return names;
}

function qualifiedKeys(localeMap) {
  const keys = new Map();
  for (const [namespace, entries] of localeMap) {
    for (const [key, value] of entries) {
      keys.set(`${namespace}.${key}`, value);
    }
  }
  return keys;
}

function checkLocaleParity() {
  const loaded = new Map(LOCALES.map((locale) => [locale, qualifiedKeys(loadLocale(locale))]));
  const reference = loaded.get(REFERENCE_LOCALE);

  for (const locale of LOCALES) {
    if (locale === REFERENCE_LOCALE) continue;
    const other = loaded.get(locale);

    for (const key of reference.keys()) {
      if (!other.has(key)) fail(`[缺失键] ${key}@${locale}`);
    }
    for (const key of other.keys()) {
      if (!reference.has(key)) fail(`[多余键] ${key}@${locale}（${REFERENCE_LOCALE} 中不存在）`);
    }

    for (const [key, value] of reference) {
      const counterpart = other.get(key);
      if (counterpart === undefined) continue;

      const refPlaceholders = [...placeholders(value)].sort();
      const otherPlaceholders = [...placeholders(counterpart)].sort();
      if (refPlaceholders.join(',') !== otherPlaceholders.join(',')) {
        fail(
          `[占位符不一致] ${key}：${REFERENCE_LOCALE} 有 [${refPlaceholders.join(', ')}]，` +
            `${locale} 有 [${otherPlaceholders.join(', ')}]`,
        );
      }

      if (value === counterpart && !SAME_VALUE_ALLOWED.has(key) && /[^\d\s\p{P}]/u.test(value)) {
        fail(
          `[疑似未撰写译文] ${key} 在 ${REFERENCE_LOCALE} 与 ${locale} 取值完全相同；` +
            '中英文文案需分别撰写（DESIGN-SYSTEM 第 9 节第 1 条）。若确应相同，请加入 SAME_VALUE_ALLOWED。',
        );
      }
    }
  }

  return loaded;
}

function collectSourceFiles(dir, sink) {
  let entries;
  try {
    entries = readdirSync(dir);
  } catch {
    return;
  }
  for (const entry of entries) {
    if (entry === 'node_modules' || entry === '.next' || entry === 'generated') continue;
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) {
      collectSourceFiles(full, sink);
    } else if (/\.(ts|tsx)$/.test(entry)) {
      sink.push(full);
    }
  }
}

/**
 * 校验源码使用的键存在。约定：调用形如 t('namespace.path') 或 tRedline('...')，
 * 键必须是字面量，禁止拼接（拼接键无法静态校验，规则上不允许）。
 */
function checkSourceKeys(loaded) {
  const files = [];
  collectSourceFiles(join(REPO_ROOT, 'apps'), files);
  collectSourceFiles(join(REPO_ROOT, 'packages', 'ui', 'src'), files);

  const known = loaded.get(REFERENCE_LOCALE);

  for (const file of files) {
    const text = readFileSync(file, 'utf8');
    const translators = new Map();
    const bindingPatterns = [
      /const\s+([A-Za-z_$][\w$]*)\s*=\s*await\s+getTranslations\(\s*'([^']+)'\s*\)/g,
      /const\s+([A-Za-z_$][\w$]*)\s*=\s*await\s+getTranslations\(\s*\{[^}]*namespace:\s*'([^']+)'[^}]*\}\s*\)/g,
      /const\s+([A-Za-z_$][\w$]*)\s*=\s*useTranslations\(\s*'([^']+)'\s*\)/g,
    ];
    for (const pattern of bindingPatterns) {
      for (const match of text.matchAll(pattern)) translators.set(match[1], match[2]);
    }
    for (const match of text.matchAll(/const\s+([A-Za-z_$][\w$]*)\s*=\s*useTranslations\(\s*\)/g)) {
      translators.set(match[1], '');
    }

    for (const match of text.matchAll(/\b([A-Za-z_$][\w$]*)\(\s*'([^']+)'\s*[),]/g)) {
      const [, translator, key] = match;
      if (!translators.has(translator)) continue;
      if (!key.includes('.')) continue;
      const namespace = translators.get(translator);
      const qualifiedKey = namespace === '' ? key : `${namespace}.${key}`;
      if (!known.has(qualifiedKey)) {
        const line = text.slice(0, match.index).split('\n').length;
        fail(
          `[源码引用了不存在的翻译键] ${relative(REPO_ROOT, file)}:${line} → ${qualifiedKey}`,
        );
      }
    }
  }
}

const loaded = checkLocaleParity();
checkSourceKeys(loaded);

if (failures.length > 0) {
  for (const message of failures) {
    process.stderr.write(`${message}\n`);
  }
  process.stderr.write(`\n翻译键门禁失败：${failures.length} 项\n`);
  process.exit(1);
}

const total = loaded.get(REFERENCE_LOCALE).size;
process.stdout.write(
  `翻译键门禁通过：${LOCALES.join(' / ')} 各 ${total} 个键，占位符一致，源码引用全部命中\n`,
);
