#!/usr/bin/env node
/**
 * 为 contracts/ts 生成物写入来源版本标记。
 * contracts/README.md 规则 3：生成物必须标注来源版本（源契约文件的 git 提交哈希）。
 */

import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

const REPO_ROOT = join(import.meta.dirname, '..');
const SOURCE_CONTRACT = 'docs/api/openapi.yaml';
const ARTIFACT = 'contracts/ts/openapi.d.ts';
const STAMP_PATH = join(REPO_ROOT, 'contracts', 'ts', 'SOURCE.md');

function gitCommitOfSource() {
  try {
    return execFileSync('git', ['log', '-1', '--format=%H', '--', SOURCE_CONTRACT], {
      cwd: REPO_ROOT,
      encoding: 'utf8',
    }).trim();
  } catch {
    return '';
  }
}

function sha256(relativePath) {
  const bytes = readFileSync(join(REPO_ROOT, relativePath));
  return createHash('sha256').update(bytes).digest('hex');
}

const commit = gitCommitOfSource();
const sourceHash = sha256(SOURCE_CONTRACT);

const lines = [
  '# contracts/ts 生成物来源',
  '',
  '| 字段 | 内容 |',
  '|---|---|',
  `| 源契约 | \`${SOURCE_CONTRACT}\` |`,
  `| 源契约最后提交 | \`${commit === '' ? '未纳入 git 历史' : commit}\` |`,
  `| 源契约内容 SHA-256 | \`${sourceHash}\` |`,
  `| 生成物 | \`${ARTIFACT}\` |`,
  '| 生成命令 | `pnpm api:generate` |',
  '| 校验命令 | `pnpm api:check` |',
  '',
  '生成物由 `openapi-typescript` 机器生成，禁止手工编辑（contracts/README.md 规则 1）。',
  '源契约变更后必须重新生成并提交，否则 CI 阶段 2 的 `pnpm api:check` 会失败。',
  '',
];

writeFileSync(STAMP_PATH, lines.join('\n'), 'utf8');
process.stdout.write(`[契约来源标记] contracts/ts/SOURCE.md（源提交 ${commit || 'n/a'}）\n`);
