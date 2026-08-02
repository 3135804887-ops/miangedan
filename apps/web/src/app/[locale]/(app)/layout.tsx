/** (app) 路由组布局：登录后应用壳（侧边导航 + 内容区）。 */

import type { ReactNode } from 'react';

import { AppNav } from '../../../components/app-nav.tsx';

export default function AppGroupLayout({
  children,
}: {
  children: ReactNode;
}): ReactNode {
  return (
    <AppNav>
      <div className="mgd-page">{children}</div>
    </AppNav>
  );
}
