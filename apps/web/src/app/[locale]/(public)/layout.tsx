/** (public) 路由组布局：落地页与登录页品牌页眉。 */

import type { ReactNode } from 'react';

import { PublicHeader } from '../../../components/public-header.tsx';

export default function PublicGroupLayout({
  children,
}: {
  children: ReactNode;
}): ReactNode {
  return (
    <div className="flex min-h-screen flex-col">
      <PublicHeader />
      <main id="main-content" className="flex-1">
        {children}
      </main>
    </div>
  );
}
