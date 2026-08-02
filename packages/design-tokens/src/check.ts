/**
 * 令牌门禁（需求 G5 第 3 条、G4 第 4 条）：
 * 1. 语义名称集合与 tokens/NAMES.lock 比对（换肤只允许改色值，不允许改语义映射）
 * 2. contrast-pairs.json 逐对对比度校验（body ≥4.5:1、large ≥3:1）
 * 3. generated/* 生成物与当前令牌重新渲染结果 diff
 * 任一失败退出码为 1。
 */

import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

import { renderArtifacts } from './build-css.ts';
import { contrastRatio, CONTRAST_THRESHOLD, roundRatio, type TextSize } from './contrast.ts';
import { GENERATED_DIR, loadTokens, semanticTokenNames, TOKENS_DIR } from './load.ts';

interface ContrastPair {
  readonly foreground: string;
  readonly background: string;
  readonly textSize: TextSize;
  readonly usage: string;
}

const failures: string[] = [];

function fail(message: string): void {
  failures.push(message);
}

function resolveColorToken(tokens: ReturnType<typeof loadTokens>, ref: string): string {
  const [group, key] = ref.split('.');
  if (group !== 'color' || key === undefined) {
    throw new Error(`对比度组合只支持 color.* 令牌引用，收到 "${ref}"`);
  }
  const leaf = tokens.color[key];
  if (leaf === undefined) {
    throw new Error(`对比度组合引用了不存在的令牌 "${ref}"`);
  }
  return leaf.value;
}

function checkNames(): void {
  const lockPath = join(TOKENS_DIR, 'NAMES.lock');
  const actual = semanticTokenNames(loadTokens());

  if (!existsSync(lockPath)) {
    fail(`[令牌语义名] 缺少 ${lockPath}；首次建立请运行 pnpm tokens:build 后提交生成的锁文件`);
    return;
  }

  const expected = readFileSync(lockPath, 'utf8')
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line.length > 0 && !line.startsWith('#'));

  const added = actual.filter((name) => !expected.includes(name));
  const removed = expected.filter((name) => !actual.includes(name));

  if (added.length > 0 || removed.length > 0) {
    fail(
      `[令牌语义名变更] 新增: ${added.join('、') || '无'}；移除: ${removed.join('、') || '无'}\n` +
        '        提示：语义映射是稳定契约，换肤只允许改色值（DESIGN-SYSTEM 第 10 节第 1 条）。\n' +
        '        若变更确属必要，请同步更新 tokens/NAMES.lock 并在 PR 说明理由。',
    );
  }
}

function checkContrast(): void {
  const tokens = loadTokens();
  const raw = readFileSync(join(TOKENS_DIR, 'contrast-pairs.json'), 'utf8');
  const parsed = JSON.parse(raw) as { pairs: readonly ContrastPair[] };

  const declared = new Set<string>();
  for (const pair of parsed.pairs) {
    declared.add(pair.foreground);
    declared.add(pair.background);

    const fg = resolveColorToken(tokens, pair.foreground);
    const bg = resolveColorToken(tokens, pair.background);
    const ratio = roundRatio(contrastRatio(fg, bg));
    const threshold = CONTRAST_THRESHOLD[pair.textSize];

    if (ratio < threshold) {
      fail(
        `[对比度不足] ${pair.foreground} on ${pair.background} = ${ratio}:1（要求 ${pair.textSize} ≥${threshold}:1，用途：${pair.usage}）\n` +
          '        建议：调整该令牌的占位色值，或将该组合改为 large 文本用途并确认实际字号。',
      );
    }
  }

  for (const key of Object.keys(tokens.color)) {
    if (!declared.has(`color.${key}`)) {
      fail(
        `[对比度未登记] color.${key} 未出现在 tokens/contrast-pairs.json 的任何组合中；` +
          '每个语义颜色至少需登记一个用途组合，否则视为未校验。',
      );
    }
  }
}

function checkGeneratedArtifacts(): void {
  const artifacts = renderArtifacts();
  for (const [fileName, expected] of Object.entries(artifacts)) {
    const target = join(GENERATED_DIR, fileName);
    if (!existsSync(target)) {
      fail(`[令牌生成物缺失] ${target}；请运行 pnpm tokens:build 并提交产物`);
      continue;
    }
    const actual = readFileSync(target, 'utf8');
    if (actual !== expected) {
      fail(
        `[令牌生成物漂移] generated/${fileName} 与 tokens/*.json 不一致；` +
          '请运行 pnpm tokens:build 并提交产物。',
      );
    }
  }
}

checkNames();
checkContrast();
checkGeneratedArtifacts();

if (failures.length > 0) {
  for (const message of failures) {
    process.stderr.write(`${message}\n`);
  }
  process.stderr.write(`\n令牌门禁失败：${failures.length} 项\n`);
  process.exit(1);
}

process.stdout.write('令牌门禁通过：语义名称锁、对比度组合、生成物 diff 全部一致\n');
