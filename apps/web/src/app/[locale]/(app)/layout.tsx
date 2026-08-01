/** (app) 路由组布局：常规页面限宽 1200px（DESIGN-SYSTEM 第 7 节）。 */

import type { ReactNode } from 'react';

export default function AppGroupLayout({ children }: { children: ReactNode }): ReactNode {
  return <div className="mgd-shell">{children}</div>;
}
