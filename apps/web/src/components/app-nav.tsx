'use client';

import {
  AppShell,
  IconBill,
  IconDashboard,
  IconFile,
  IconHome,
  IconLogout,
  IconOrg,
  IconSettings,
  type NavGroup,
} from '@mgd/ui';
import { useLocale, useTranslations } from 'next-intl';
import { usePathname } from 'next/navigation';
import type { ReactNode } from 'react';

export function BrandMark(): React.ReactNode {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24" className="size-5" fill="none" stroke="currentColor" strokeWidth={2}>
      <path d="M12 2.5 21 7v5.5c0 5-3.8 8.6-9 10.5-5.2-1.9-9-5.5-9-10.5V7z" strokeLinejoin="round" />
      <path d="M8.5 12.2 11 14.5l4.5-5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

export function AppNav({
  orgMode = false,
  children,
}: {
  orgMode?: boolean;
  children?: ReactNode;
}): React.ReactNode {
  const t = useTranslations('common');
  const locale = useLocale();
  const pathname = usePathname();
  const isActive = (prefix: string) => pathname === prefix || pathname.startsWith(`${prefix}/`);
  const href = (path: string) => `/${locale}${path}`;

  const groups: readonly NavGroup[] = [
    {
      items: [
        { href: href('/'), label: t('nav.landing'), icon: <IconHome size={19} />, active: pathname === href('/') },
        { href: href('/dashboard'), label: t('nav.dashboard'), icon: <IconDashboard size={19} />, active: isActive(href('/dashboard')) },
        { href: href('/library'), label: t('nav.library'), icon: <IconFile size={19} />, active: isActive(href('/library')) },
        { href: href('/billing'), label: t('nav.billing'), icon: <IconBill size={19} />, active: isActive(href('/billing')) },
        { href: href('/settings'), label: t('nav.settings'), icon: <IconSettings size={19} />, active: isActive(href('/settings')) },
      ],
    },
    ...(orgMode
      ? [
          {
            label: t('nav.primaryLabel'),
            items: [
              { href: href('/org/demo/assignments'), label: t('org.navAssignments'), icon: <IconOrg size={19} />, active: isActive(href('/org/demo/assignments')) },
              { href: href('/org/demo/members'), label: t('org.navMembers'), icon: <IconOrg size={19} />, active: isActive(href('/org/demo/members')) },
              { href: href('/org/demo/aggregates'), label: t('org.navAggregates'), icon: <IconChart size={19} />, active: isActive(href('/org/demo/aggregates')) },
              { href: href('/org/demo/permissions'), label: t('org.navPermissions'), icon: <IconShield size={19} />, active: isActive(href('/org/demo/permissions')) },
              { href: href('/org/demo/shares'), label: t('org.navShares'), icon: <IconLock size={19} />, active: isActive(href('/org/demo/shares')) },
            ],
          },
        ]
      : []),
  ];

  return (
    <AppShell
      brand={t('brand.name')}
      brandMark={<BrandMark />}
      groups={groups}
      topbar={
        <div className="flex w-full items-center justify-between gap-3">
          <span className="text-sm text-neutral-600">{t('brand.tagline')}</span>
          <div className="flex items-center gap-2">
            <a href={href('/settings')} className="mgd-target-min grid size-10 place-items-center rounded-lg text-neutral-600 hover:bg-surface-muted" aria-label={t('nav.settings')}>
              <IconSettings size={18} />
            </a>
            <a href={href('/auth')} className="mgd-target-min grid size-10 place-items-center rounded-lg text-neutral-600 hover:bg-surface-muted" aria-label="退出">
              <IconLogout size={18} />
            </a>
          </div>
        </div>
      }
    >
      {children}
    </AppShell>
  );
}

function IconChart(props: { size?: number }): React.ReactNode {
  return (
    <svg aria-hidden="true" width={props.size} height={props.size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
      <path d="M4 20V10M10 20V4M16 20v-8M22 20H2" />
    </svg>
  );
}

function IconShield(props: { size?: number }): React.ReactNode {
  return (
    <svg aria-hidden="true" width={props.size} height={props.size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 3 5 6v5c0 5 3 8.5 7 10 4-1.5 7-5 7-10V6z" />
      <path d="m9 12 2 2 4-4" />
    </svg>
  );
}

function IconLock(props: { size?: number }): React.ReactNode {
  return (
    <svg aria-hidden="true" width={props.size} height={props.size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
      <rect x="5" y="11" width="14" height="9" rx="2" />
      <path d="M8 11V8a4 4 0 0 1 8 0v3" />
    </svg>
  );
}
