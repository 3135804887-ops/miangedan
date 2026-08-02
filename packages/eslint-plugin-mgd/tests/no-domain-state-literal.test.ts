/**
 * no-domain-state-literal 规则的正例 / 反例，以及内嵌清单与 @mgd/domain-states 的漂移断言。
 */

import { RuleTester } from '@typescript-eslint/rule-tester';
import { PROJECT_STATUSES, SESSION_STATUSES } from '@mgd/domain-states';
import { afterAll, describe, expect, it } from 'vitest';

import rule, {
  KNOWN_PROJECT_STATUSES,
  KNOWN_SESSION_STATUSES,
} from '../rules/no-domain-state-literal.mjs';

RuleTester.afterAll = afterAll;
RuleTester.describe = describe;
RuleTester.it = it;

const ruleTester = new RuleTester();

ruleTester.run('no-domain-state-literal', rule as never, {
  valid: [
    // 从共享模块导入是唯一正确用法
    { code: "import { PROJECT_STATUSES } from '@mgd/domain-states';" },
    // 白名单：机构可见性投影，不是面试状态
    { code: "const progress = 'COMPLETED_OR_EXITED';" },
    { code: "const conclusion = 'PASS';" },
    // 普通文案与小写标识不受影响
    { code: "const label = '恭喜你通过本轮面试，已进入下一轮';" },
    { code: "const key = 'scr10.result.pass';" },
    // 单段大写但非状态名，不误报
    { code: "const unit = 'CNY';" },
  ],
  invalid: [
    {
      code: "const status = 'ROUND_PASSED';",
      errors: [{ messageId: 'knownState' }],
    },
    {
      code: "const status = 'LIVE';",
      errors: [{ messageId: 'knownState' }],
    },
    {
      code: "if (session === 'PAUSED_SYSTEM') { doSomething(); }",
      errors: [{ messageId: 'knownState' }],
    },
    {
      // 拼写错误：不在枚举内，按疑似自创状态名报错
      code: "const status = 'ROUND_PASSD';",
      errors: [{ messageId: 'suspectedState' }],
    },
    {
      // 自创状态名
      code: "const status = 'ROUND_TIMED_OUT';",
      errors: [{ messageId: 'suspectedState' }],
    },
    {
      // 字面量类型位置同样禁止
      code: "type S = 'MATERIAL_REVIEW';",
      errors: [{ messageId: 'knownState' }],
    },
    {
      code: 'const status = `EVALUATION_INCOMPLETE`;',
      errors: [{ messageId: 'knownState' }],
    },
  ],
});

describe('规则内嵌状态清单与 @mgd/domain-states 一致', () => {
  it('项目状态清单无漂移', () => {
    expect([...KNOWN_PROJECT_STATUSES].sort()).toEqual([...PROJECT_STATUSES].sort());
  });

  it('会话状态清单无漂移', () => {
    expect([...KNOWN_SESSION_STATUSES].sort()).toEqual([...SESSION_STATUSES].sort());
  });
});
