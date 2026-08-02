#!/usr/bin/env node
/**
 * 契约生成物 diff 校验（需求 G6 第 2 条）。
 * 重新生成到临时文件后与仓库内已提交产物比对；存在差异即失败。
 */

import { execFileSync } from 'node:child_process';
import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const REPO_ROOT = join(import.meta.dirname, '..');
const SOURCE_CONTRACT = join(REPO_ROOT, 'docs', 'api', 'openapi.yaml');
const COMMITTED = join(REPO_ROOT, 'contracts', 'ts', 'openapi.d.ts');

const workDir = mkdtempSync(join(tmpdir(), 'mgd-contract-'));
const regenerated = join(workDir, 'openapi.d.ts');

try {
  execFileSync(
    process.execPath,
    [
      join(REPO_ROOT, 'node_modules', 'openapi-typescript', 'bin', 'cli.js'),
      SOURCE_CONTRACT,
      '-o',
      regenerated,
    ],
    { cwd: REPO_ROOT, stdio: ['ignore', 'ignore', 'inherit'] },
  );

  let committed;
  try {
    committed = readFileSync(COMMITTED, 'utf8');
  } catch {
    process.stderr.write(
      '[契约生成物缺失] contracts/ts/openapi.d.ts 不存在；请运行 pnpm api:generate 并提交产物。\n',
    );
    process.exit(1);
  }

  const fresh = readFileSync(regenerated, 'utf8');

  if (committed !== fresh) {
    process.stderr.write(
      '[契约生成物漂移] contracts/ts/openapi.d.ts 与 docs/api/openapi.yaml 不一致，' +
        '请运行 pnpm api:generate 并提交产物。\n',
    );
    process.exit(1);
  }

  process.stdout.write('契约生成物校验通过：contracts/ts/openapi.d.ts 与源契约一致\n');
} finally {
  rmSync(workDir, { recursive: true, force: true });
}
