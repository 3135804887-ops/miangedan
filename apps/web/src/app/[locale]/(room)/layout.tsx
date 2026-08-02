/** (room) 路由组布局：实时面试房间全宽利用（DESIGN-SYSTEM 第 7 节）。 */

import type { ReactNode } from 'react';

export default function RoomGroupLayout({ children }: { children: ReactNode }): ReactNode {
  return <div className="mgd-shell mgd-shell--full">{children}</div>;
}
