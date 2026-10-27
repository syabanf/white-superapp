"use client";

import * as React from "react";
import Link from "next/link";
import { useParams, usePathname } from "next/navigation";
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
import { activeNavHref, adminNav, clientNav, erpNav, generalNav, isActivePath, workspaceOf } from "@/components/shell/nav-config";
import type { SwitcherClient } from "@/components/shell/client-switcher";
import { cn } from "@/lib/utils";
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
    } else if (workspaceOf(pathname) === "erp") {
      // ERP: the module, then the page inside it ("SDM › Rekrutmen").
      const items = erpNav.flatMap((g) => g.items);
      const current = activeNavHref(pathname, items);
      const top = items.find((it) => it.href === current || it.children?.some((c) => c.href === current));
      const child = top?.children?.find((c) => c.href === current);
      const group = erpNav.find((g) => top && g.items.includes(top));
      if (group && group.label !== top?.label) out.push({ label: group.label });
      out.push({ label: top?.label ?? t.erp.title, href: child ? top?.href : undefined });
      if (child) out.push({ label: child.label });
    } else {
      const item = [...generalNav, ...adminNav].find((n) => isActivePath(pathname, n.href, n.exact));
      out.push({ label: item?.label ?? t.app.name });
    }
    return out;
  }, [pathname, client]);

  return (
    <header className="sticky top-0 z-30 h-16 shrink-0 bg-background md:h-[4.5rem] md:rounded-t-[2rem]">
      {/* Same container as the page, so the trigger lines up with the page title below */}
      <div className="page-container flex h-full items-center gap-2.5">
      <SidebarTrigger className="hidden size-10 shrink-0 rounded-2xl bg-card shadow-(--card-shadow) hover:bg-muted md:inline-flex" />
      {/* The trail sits in a soft pill, like a search field */}
      <Breadcrumb className="min-w-0 flex-1 overflow-hidden md:flex md:h-10 md:max-w-xl md:items-center md:rounded-full md:bg-card md:px-4 md:shadow-(--card-shadow)">
        <BreadcrumbList className="flex-nowrap overflow-hidden">
          {crumbs.map((c, i) => {
            const last = i === crumbs.length - 1;
            return (
              <React.Fragment key={`${c.label}-${i}`}>
                <BreadcrumbItem
                  className={cn(
                    "min-w-0",
                    // Phones and tablets show the current page only; the trail returns from lg up.
                    !last && "hidden lg:inline-flex",
                    i === 0 && crumbs.length > 2 && "lg:hidden 2xl:inline-flex",
                  )}
                >
                  {last || !c.href ? (
                    <BreadcrumbPage className="truncate" title={c.label}>
                      {c.label}
                    </BreadcrumbPage>
                  ) : (
                    <BreadcrumbLink asChild>
                      <Link href={c.href} className="truncate" title={c.label}>
                        {c.label}
                      </Link>
                    </BreadcrumbLink>
                  )}
                </BreadcrumbItem>
                {!last ? (
                  <BreadcrumbSeparator
                    className={cn("hidden lg:block", i === 0 && crumbs.length > 2 && "lg:hidden 2xl:block")}
                  />
                ) : null}
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
      </div>
    </header>
  );
}
