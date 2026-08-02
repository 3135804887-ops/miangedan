/**
 * 编译期契约断言（需求 G1 第 1 条）。
 *
 * 本文件不产生运行时行为，只让 `tsc --noEmit` 在以下任一情况失败：
 *   - 本包枚举与 docs/api/openapi.yaml 生成的类型出现任何增减或拼写差异
 *   - openapi 契约变更后未同步本包枚举
 *
 * 便利设置在 openapi 中是路径内联枚举，不便做类型引用，改由测试期读取
 * ai/schemas/turn-evidence.schema.json 断言（见 tests/accommodations.contract.test.ts）。
 */

import type { components } from '@mgd/api-types';

import type { ProjectStatus } from './project.ts';
import type { SessionStatus } from './session.ts';

/** A 与 B 必须互为子集，即完全相等；不等时类型为 never，赋值 true 报错。 */
type Exact<A, B> = [A] extends [B] ? ([B] extends [A] ? true : never) : never;

// docs/api/openapi.yaml → components.schemas.ProjectStatus（15 项）
const projectStatusMatchesContract: Exact<
  ProjectStatus,
  components['schemas']['ProjectStatus']
> = true;

// docs/api/openapi.yaml → components.schemas.Session.room_status（10 项）
const sessionStatusMatchesContract: Exact<
  SessionStatus,
  NonNullable<components['schemas']['Session']['room_status']>
> = true;

void projectStatusMatchesContract;
void sessionStatusMatchesContract;
