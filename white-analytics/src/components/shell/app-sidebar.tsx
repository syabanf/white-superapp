"use client";

import * as React from "react";
import Link from "next/link";
import { useParams, usePathname, useSearchParams } from "next/navigation";
import { ChevronRight, FilePlus2, Plus } from "lucide-react";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
  useSidebar,
} from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import { MixistLogo } from "@/components/brand/logo";
import { ClientSwitcher, type SwitcherClient } from "@/components/shell/client-switcher";
import { UserMenu, type ShellUser } from "@/components/shell/user-menu";
import {
  adminNav,
  clientNav,
  generalNav,
  activeNavHref,
  visibleErpNav,
  workspaceOf,
  type ErpAccess,
  type NavItem,
} from "@/components/shell/nav-config";
import { WorkspaceSwitcher } from "@/components/shell/workspace-switcher";
import { cn } from "@/lib/utils";
import { t } from "@/i18n/id";

export function AppSidebar({
  clients,
  user,
  erpAccess,
}: {
  clients: SwitcherClient[];
  user: ShellUser;
  erpAccess: ErpAccess | null;
}) {
  const pathname = usePathname();
  const inErp = workspaceOf(pathname) === "erp";
  const erpGroups = visibleErpNav(erpAccess);
  const searchParams = useSearchParams();
  const params = useParams<{ slug?: string }>();
  const slug = typeof params?.slug === "string" ? params.slug : undefined;
  const currentClient = slug ? clients.find((c) => c.slug === slug) : undefined;
  const { state } = useSidebar();
  const collapsed = state === "collapsed";
  const withContext = React.useCallback(
    (href: string) => {
      const query = new URLSearchParams();
      for (const key of ["from", "to", "preset", "compare"]) {
        const value = searchParams.get(key);
        if (value) query.set(key, value);
      }
      const suffix = query.toString();
      return suffix ? `${href}?${suffix}` : href;
    },
    [searchParams],
  );

  return (
    <Sidebar collapsible="icon" variant="floating" className="border-r-0">
      <SidebarHeader className="gap-3 px-4 pt-6 group-data-[collapsible=icon]:items-center group-data-[collapsible=icon]:px-2">
        <Link
          href="/"
          className="mb-1 flex h-9 items-center gap-2 rounded-xl px-1 outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <MixistLogo variant={collapsed ? "mark" : "full"} size={collapsed ? 28 : 36} />
        </Link>
        <WorkspaceSwitcher collapsed={collapsed} showErp={erpGroups.length > 1} />
        {!inErp ? <ClientSwitcher clients={clients} current={currentClient} collapsed={collapsed} /> : null}
      </SidebarHeader>

      <SidebarContent className="scrollbar-thin px-1 pt-2">
        {inErp ? (
          erpGroups.map((g) => (
            <NavGroup
              key={g.label}
              label={g.label}
              items={g.items}
              pathname={pathname}
              withContext={(href) => href}
            />
          ))
        ) : (
          <MarketingNav
            currentClient={currentClient}
            isAdmin={user.role === "ADMIN"}
            pathname={pathname}
            withContext={withContext}
          />
        )}
      </SidebarContent>

      <SidebarFooter className="gap-3 p-3 pb-4">
        {!collapsed && !inErp ? (
          <Button asChild className="h-11 w-full justify-center">
            {currentClient ? (
              <Link href={withContext(`/clients/${currentClient.slug}/reports`)}>
                <FilePlus2 className="size-4" /> {t.reports.builder}
              </Link>
            ) : (
              <Link href="/clients/new">
                <Plus className="size-4" /> {t.clients.new}
              </Link>
            )}
          </Button>
        ) : null}
        <UserMenu user={user} collapsed={collapsed} />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}

function MarketingNav({
  currentClient,
  isAdmin,
  pathname,
  withContext,
}: {
  currentClient?: SwitcherClient;
  isAdmin: boolean;
  pathname: string;
  withContext: (href: string) => string;
}) {
  return (
    <>
      <NavGroup label={t.nav.general} items={generalNav} pathname={pathname} withContext={withContext} />
      {currentClient ? (
        <NavGroup
          label={currentClient.name}
          items={clientNav(currentClient.slug)}
          pathname={pathname}
          withContext={withContext}
        />
      ) : null}
      {isAdmin ? (
        <NavGroup label={t.nav.admin} items={adminNav} pathname={pathname} withContext={withContext} />
      ) : null}
    </>
  );
}

/** Soft icon chip: quiet on idle rows, brand-tinted when the row is active. */
function NavIcon({ icon: Icon, active }: { icon: NavItem["icon"]; active: boolean }) {
  return (
    <span
      aria-hidden
      className={cn(
        "flex size-8 shrink-0 items-center justify-center rounded-xl transition-colors duration-200",
        active
          ? "bg-[color-mix(in_oklab,var(--brand)_14%,transparent)] text-brand"
          : "text-muted-foreground group-hover/row:bg-muted group-hover/row:text-foreground",
      )}
    >
      <Icon className="size-[18px]" strokeWidth={active ? 2.25 : 1.75} />
    </span>
  );
}

/** Accent bar on the panel edge beside the active row. */
function ActiveBar() {
  return (
    <span
      aria-hidden
      className="pointer-events-none absolute top-2.5 -left-3 h-6 w-1 rounded-r-full bg-brand group-data-[collapsible=icon]:-left-2"
    />
  );
}

const ROW =
  "group/row h-11 gap-3 rounded-2xl px-1.5 font-normal text-muted-foreground transition-colors duration-150 hover:bg-transparent hover:text-foreground data-[active=true]:bg-transparent data-[active=true]:font-semibold data-[active=true]:text-foreground group-data-[collapsible=icon]:size-11! group-data-[collapsible=icon]:p-1.5!";

function NavGroup({
  label,
  items,
  pathname,
  withContext,
}: {
  label: string;
  items: NavItem[];
  pathname: string;
  withContext: (href: string) => string;
}) {
  // Accordion: the section you are in is open, and opening another closes it.
  // A manual choice only lasts until the next navigation, then the route decides again.
  const current = activeNavHref(pathname, items);
  const routeOpen =
    items.find((it) => it.children && (it.href === current || it.children.some((c) => c.href === current)))?.href ??
    null;
  const [manual, setManual] = React.useState<{ at: string; href: string | null } | null>(null);
  const openHref = manual?.at === pathname ? manual.href : routeOpen;

  return (
    <SidebarGroup className="px-2 py-1.5">
      <SidebarGroupLabel className="label-mono h-auto truncate px-2 pb-1.5 text-muted-foreground">
        {label}
      </SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu className="gap-0.5">
          {items.map((item) => {
            const active = item.href === current;
            const childActive = item.children?.some((c) => c.href === current) ?? false;
            const open = openHref === item.href;
            return (
              <SidebarMenuItem key={item.href}>
                {active || childActive ? <ActiveBar /> : null}
                <SidebarMenuButton
                  asChild
                  tooltip={item.label}
                  isActive={active || childActive}
                  className={cn(ROW, item.children && "pr-9")}
                >
                  <Link href={withContext(item.href)}>
                    <NavIcon icon={item.icon} active={active || childActive} />
                    <span>{item.label}</span>
                  </Link>
                </SidebarMenuButton>

                {item.children?.length ? (
                  <>
                    <button
                      type="button"
                      aria-expanded={open}
                      aria-label={`${open ? t.nav.collapse : t.nav.expand} ${item.label}`}
                      onClick={() => setManual({ at: pathname, href: open ? null : item.href })}
                      className="absolute top-2 right-1 flex size-7 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground group-data-[collapsible=icon]:hidden"
                    >
                      <ChevronRight
                        className={cn("size-3.5 transition-transform duration-200", open && "rotate-90")}
                      />
                    </button>
                    {open ? (
                      <ul className="mt-0.5 mb-2 ml-[1.15rem] space-y-0.5 border-l border-border/70 py-0.5 pl-3 group-data-[collapsible=icon]:hidden">
                        {item.children.map((child) => {
                          const on = child.href === current;
                          return (
                            <li key={child.href}>
                              <Link
                                href={withContext(child.href)}
                                aria-current={on ? "page" : undefined}
                                className={cn(
                                  "flex h-8 items-center gap-2.5 rounded-full px-3 text-[13px] outline-none transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-ring",
                                  on
                                    ? "bg-[color-mix(in_oklab,var(--brand)_9%,transparent)] font-medium text-foreground"
                                    : "text-muted-foreground hover:bg-muted hover:text-foreground",
                                )}
                              >
                                <span
                                  aria-hidden
                                  className={cn(
                                    "size-1.5 shrink-0 rounded-full",
                                    on ? "bg-brand" : "bg-border",
                                  )}
                                />
                                <span className="truncate">{child.label}</span>
                              </Link>
                            </li>
                          );
                        })}
                      </ul>
                    ) : null}
                  </>
                ) : null}
              </SidebarMenuItem>
            );
          })}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
