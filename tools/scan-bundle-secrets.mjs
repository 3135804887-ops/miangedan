#!/usr/bin/env node
/**
 * 打包产物密钥扫描（需求 G8 第 3 条）。
 * 在 CI 阶段 6 构建之后运行：前端产物不得包含任何供应商密钥、令牌或凭证。
 */

import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';

const REPO_ROOT = join(import.meta.dirname, '..');
const SCAN_TARGETS = [
  join(REPO_ROOT, 'apps', 'web', '.next'),
  join(REPO_ROOT, 'apps', 'admin', '.next'),
];

/** 与 tools/validate_docs.py 的 secrets 套件同源的模式集。 */
const SECRET_PATTERNS = [
  { name: 'AWS Access Key', re: /\bAKIA[0-9A-Z]{16}\b/ },
  { name: 'Private Key Block', re: /-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----/ },
  { name: 'Slack Token', re: /\bxox[abprs]-[0-9A-Za-z-]{10,}/ },
  { name: 'GitHub Token', re: /\bgh[pousr]_[0-9A-Za-z]{36,}/ },
  { name: 'OpenAI-style Key', re: /\bsk-[A-Za-z0-9]{32,}\b/ },
  { name: 'Google API Key', re: /\bAIza[0-9A-Za-z_-]{35}\b/ },
  { name: 'JWT', re: /\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b/ },
];

const SCANNABLE = /\.(js|mjs|cjs|json|css|html|txt|map)$/;
const failures = [];

function walk(dir, onFile) {
  let entries;
  try {
    entries = readdirSync(dir);
  } catch {
    return;
  }
  for (const entry of entries) {
    if (entry === 'cache') continue;
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) {
      walk(full, onFile);
    } else {
      onFile(full);
    }
  }
}

let scanned = 0;

for (const target of SCAN_TARGETS) {
  walk(target, (file) => {
    if (!SCANNABLE.test(file)) return;
    scanned += 1;
    const text = readFileSync(file, 'utf8');
    for (const pattern of SECRET_PATTERNS) {
      const match = pattern.re.exec(text);
      if (match !== null) {
        const line = text.slice(0, match.index).split('\n').length;
        failures.push(
          `[打包产物疑似密钥] ${relative(REPO_ROOT, file)}:${line} 命中 ${pattern.name}`,
        );
      }
    }
  });
}

if (scanned === 0) {
  process.stderr.write(
    '[打包产物缺失] 未找到 apps/*/.next 产物；请先运行 pnpm build 再执行本扫描。\n',
  );
  process.exit(1);
}

if (failures.length > 0) {
  for (const message of failures) {
    process.stderr.write(`${message}\n`);
  }
  process.stderr.write(`\n打包产物密钥扫描失败：${failures.length} 项\n`);
  process.exit(1);
}

process.stdout.write(`打包产物密钥扫描通过：已扫描 ${scanned} 个文件，0 项命中\n`);
