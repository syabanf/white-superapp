"use client";

import * as React from "react";
import Link from "next/link";
import { useParams, usePathname } from "next/navigation";
import { Separator } from "@/components/ui/separator";
import { SidebarTrigger } from "@/components/ui/sidebar";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { DateRangePicker } from "@/components/shell/date-range-picker";
import { HelpButton } from "@/components/shell/help-button";
import { NotificationBell } from "@/components/shell/notification-bell";
import type { NotificationFeed } from "@/features/notifications/queries";
import { clientNav, generalNav, adminNav, isActivePath } from "@/components/shell/nav-config";
import type { SwitcherClient } from "@/components/shell/client-switcher";
import { t } from "@/i18n/id";

/** Pages where the global date-range filter is meaningful. */
function usesDateRange(pathname: string): boolean {
  if (pathname === "/") return true;
  if (!pathname.startsWith("/clients/")) return false;
  // Setup surfaces have nothing to filter by date.
  if (pathname === "/clients/new") return false;
  if (/\/clients\/[^/]+\/settings/.test(pathname)) return false;
  // Publishing has its own calendar navigation; research/backlinks are point-in-time.
  if (/\/clients\/[^/]+\/publish(\/|$)/.test(pathname)) return false;
  if (/\/clients\/[^/]+\/seo\/(research|backlinks|competitors)(\/|$)/.test(pathname)) return false;
  return true;
}

export function Topbar({ clients, notifications }: { clients: SwitcherClient[]; notifications: NotificationFeed }) {
  const pathname = usePathname();
  const params = useParams<{ slug?: string }>();
  const slug = typeof params?.slug === "string" ? params.slug : undefined;
  const client = slug ? clients.find((c) => c.slug === slug) : undefined;

  const crumbs = React.useMemo(() => {
    const out: { label: string; href?: string }[] = [];
    if (pathname === "/clients/new") {
      return [{ label: t.nav.clients, href: "/clients" }, { label: t.clients.new }];
    }
    if (client) {
      out.push({ label: t.nav.clients, href: "/clients" });
      const nav = clientNav(client.slug);
      const top = nav.find((n) => isActivePath(pathname, n.href, n.exact));
      out.push({ label: client.name, href: top && top.href !== `/clients/${client.slug}` ? `/clients/${client.slug}` : undefined });
      if (top && top.href !== `/clients/${client.slug}`) {
        const child = top.children?.find((c) => isActivePath(pathname, c.href));
        out.push({ label: top.label, href: child ? top.href : undefined });
        if (child) out.push({ label: child.label });
      }
    } else {
      const item = [...generalNav, ...adminNav].find((n) => isActivePath(pathname, n.href, n.exact));
      out.push({ label: item?.label ?? t.app.name });
    }
    return out;
  }, [pathname, client]);

  return (
    <header className="sticky top-0 z-30 flex h-14 shrink-0 items-center gap-2 border-b bg-canvas/90 px-4 backdrop-blur supports-[backdrop-filter]:bg-canvas/75">
      <SidebarTrigger className="-ml-1 shrink-0" />
      <Separator orientation="vertical" className="mr-2 shrink-0 data-[orientation=vertical]:h-4" />
      <Breadcrumb className="min-w-0 flex-1 overflow-hidden">
        <BreadcrumbList className="flex-nowrap overflow-hidden">
          {crumbs.map((c, i) => {
            const last = i === crumbs.length - 1;
            return (
              <React.Fragment key={`${c.label}-${i}`}>
                <BreadcrumbItem className={i === 0 && crumbs.length > 2 ? "hidden md:inline-flex" : undefined}>
                  {last || !c.href ? (
                    <BreadcrumbPage className="truncate">{c.label}</BreadcrumbPage>
                  ) : (
                    <BreadcrumbLink asChild>
                      <Link href={c.href} className="truncate">
                        {c.label}
                      </Link>
                    </BreadcrumbLink>
                  )}
                </BreadcrumbItem>
                {!last ? <BreadcrumbSeparator className={i === 0 && crumbs.length > 2 ? "hidden md:block" : undefined} /> : null}
              </React.Fragment>
            );
          })}
        </BreadcrumbList>
      </Breadcrumb>
      <div className="ml-auto flex shrink-0 items-center gap-2">
        {usesDateRange(pathname) ? <DateRangePicker /> : null}
        <NotificationBell feed={notifications} />
        <HelpButton />
      </div>
    </header>
  );
}
