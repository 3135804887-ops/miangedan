/**
 * ESLint 规则：no-domain-state-literal（需求 G1 第 3 条）
 *
 * 禁止在应用与组件代码中书写领域状态字面量，必须从 @mgd/domain-states 导入。
 * 同时能抓住拼写错误（如 ROUND_PASSD 既不在枚举内也不在白名单内）。
 *
 * 判定范围：
 *   - 命中已知状态名（项目态 15 / 会话态 10）
 *   - 或形如 UPPER_SNAKE_CASE 的多段大写字面量（疑似自创状态名）
 *
 * 例外：
 *   - import / export 语句中的模块说明符
 *   - 白名单 ALLOWED_UPPER_LITERALS（机构可见性投影、评分结论等非面试状态枚举）
 *
 * 本文件内嵌的状态清单由 packages/eslint-plugin-mgd/tests/rule-lists.test.ts
 * 与 @mgd/domain-states 做一致性断言，防止两处漂移。
 */

export const KNOWN_PROJECT_STATUSES = [
  'DRAFT',
  'PARSING',
  'MATERIAL_REVIEW',
  'PARSE_FAILED',
  'PLAN_GENERATING',
  'PLAN_REVIEW',
  'PLAN_FAILED',
  'READY',
  'IN_SESSION',
  'SCORING',
  'ROUND_PASSED',
  'ROUND_FAILED',
  'PRACTICING',
  'EVALUATION_INCOMPLETE',
  'COMPLETED',
];

export const KNOWN_SESSION_STATUSES = [
  'ROOM_CREATED',
  'PRE_CHECK',
  'AVATAR_CONNECTING',
  'LIVE',
  'PAUSED_SYSTEM',
  'RECONNECTING',
  'DOWNGRADE_PROMPTED',
  'TEXT_DEGRADED',
  'AUTH_PAUSED',
  'ENDED',
];

/**
 * 白名单：非面试状态机的大写枚举值。
 * - 机构可见性投影（SCREEN-SPEC 第 7 节）：机构默认最小可见，且不含失败值
 * - 评分结论（openapi ResultStatus）：报告只读展示用
 * - 报告类型与数据区代码
 */
export const ALLOWED_UPPER_LITERALS = [
  'NOT_STARTED',
  'IN_PROGRESS',
  'COMPLETED_OR_EXITED',
  'PASS',
  'FAIL',
];

const MULTI_SEGMENT_UPPER = /^[A-Z][A-Z0-9]*(_[A-Z0-9]+)+$/;

const DOMAIN_STATES = new Set([...KNOWN_PROJECT_STATUSES, ...KNOWN_SESSION_STATUSES]);
const ALLOWED = new Set(ALLOWED_UPPER_LITERALS);

/** @type {import('eslint').Rule.RuleModule} */
const rule = {
  meta: {
    type: 'problem',
    docs: {
      description:
        '禁止书写面试领域状态字面量，必须从 @mgd/domain-states 导入（需求 G1：页面层不得自创状态名）',
    },
    schema: [
      {
        type: 'object',
        properties: {
          additionalAllowed: {
            type: 'array',
            items: { type: 'string' },
          },
        },
        additionalProperties: false,
      },
    ],
    messages: {
      knownState:
        '不得在页面层书写领域状态字面量 "{{value}}"；请从 @mgd/domain-states 导入对应常量。',
      suspectedState:
        '发现疑似自创状态名 "{{value}}"；它不在 @mgd/domain-states 的枚举内。若确为新状态，先更新 docs/domain/INTERVIEW-STATE-MACHINE.md 并同步枚举；若不是状态，请加入规则白名单并说明理由。',
    },
  },

  create(context) {
    const options = context.options[0] ?? {};
    const extraAllowed = new Set(options.additionalAllowed ?? []);

    function isExempt(node) {
      const parent = node.parent;
      if (parent === undefined || parent === null) return false;
      // import 'x' / export from 'x' / import('x') 的模块说明符
      if (
        parent.type === 'ImportDeclaration' ||
        parent.type === 'ExportNamedDeclaration' ||
        parent.type === 'ExportAllDeclaration' ||
        parent.type === 'ImportExpression'
      ) {
        return true;
      }
      // TS 字面量类型位置（如 type X = 'LIVE'）同样禁止，不豁免
      return false;
    }

    function check(node, value) {
      if (typeof value !== 'string') return;
      if (ALLOWED.has(value) || extraAllowed.has(value)) return;
      if (isExempt(node)) return;

      if (DOMAIN_STATES.has(value)) {
        context.report({ node, messageId: 'knownState', data: { value } });
        return;
      }

      if (MULTI_SEGMENT_UPPER.test(value)) {
        context.report({ node, messageId: 'suspectedState', data: { value } });
      }
    }

    return {
      Literal(node) {
        check(node, node.value);
      },
      TemplateLiteral(node) {
        if (node.expressions.length > 0) return;
        const raw = node.quasis[0]?.value.cooked;
        check(node, raw);
      },
    };
  },
};

export default rule;
