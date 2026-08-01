/**
 * @mgd/eslint-plugin-mgd：面个蛋前端本地 ESLint 规则集。
 * 以 .mjs 提供，避免为 lint 插件引入额外构建步骤。
 */

import noDomainStateLiteral from './rules/no-domain-state-literal.mjs';

const plugin = {
  meta: {
    name: 'mgd',
    version: '0.0.0',
  },
  rules: {
    'no-domain-state-literal': noDomainStateLiteral,
  },
};

export default plugin;
