/** Mock_Layer 环境标记（独立模块：测试/Node 环境不得加载 msw/browser）。 */

export const MOCK_READY_EVENT = 'mgd:mocks-ready';

/** 演示模式是否启用（构建期内联常量；生产默认关闭）。 */
export function isMocksEnabled(): boolean {
  return process.env.NEXT_PUBLIC_MGD_MOCKS === 'on';
}
