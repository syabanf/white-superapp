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
import { WhiteLogo } from "@/components/brand/logo";
import { ClientSwitcher, type SwitcherClient } from "@/components/shell/client-switcher";
import { UserMenu, type ShellUser } from "@/components/shell/user-menu";
import { adminNav, clientNav, generalNav, isActivePath, type NavItem } from "@/components/shell/nav-config";
import { cn } from "@/lib/utils";
import { t } from "@/i18n/id";

export function AppSidebar({ clients, user }: { clients: SwitcherClient[]; user: ShellUser }) {
  const pathname = usePathname();
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
    <Sidebar collapsible="icon" variant="sidebar" className="border-r-0">
      <SidebarHeader className="gap-4 px-4 pt-5">
        <Link
          href="/"
          className="flex h-8 items-center gap-2 outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <WhiteLogo variant={collapsed ? "mark" : "full"} size={collapsed ? 22 : 26} />
        </Link>
        <ClientSwitcher clients={clients} current={currentClient} collapsed={collapsed} />
      </SidebarHeader>

      <SidebarContent className="scrollbar-thin px-1">
        <NavGroup label={t.nav.general} items={generalNav} pathname={pathname} withContext={withContext} />
        {currentClient ? (
          <NavGroup
            label={currentClient.name}
            items={clientNav(currentClient.slug)}
            pathname={pathname}
            withContext={withContext}
          />
        ) : null}
        {user.role === "ADMIN" ? (
          <NavGroup label={t.nav.admin} items={adminNav} pathname={pathname} withContext={withContext} />
        ) : null}
      </SidebarContent>

      <SidebarFooter className="gap-3 p-3">
        {!collapsed ? (
          <Button asChild className="h-10 w-full justify-center">
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

/** Round icon chip: white on the grey rail, brand blue when the row is active. */
function NavIcon({ icon: Icon, active }: { icon: NavItem["icon"]; active: boolean }) {
  return (
    <span
      aria-hidden
      className={cn(
        "flex size-8 shrink-0 items-center justify-center rounded-full transition-colors duration-200",
        active
          ? "bg-brand text-white"
          : "bg-card text-muted-foreground shadow-(--card-shadow) group-hover/row:text-foreground",
      )}
    >
      <Icon className="size-4" />
    </span>
  );
}

const ROW =
  "group/row h-11 gap-3 rounded-full px-1.5 font-normal text-muted-foreground transition-colors duration-150 hover:bg-transparent hover:text-foreground data-[active=true]:bg-transparent data-[active=true]:font-medium data-[active=true]:text-foreground group-data-[collapsible=icon]:size-11! group-data-[collapsible=icon]:p-1.5!";

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
  const routeOpen =
    items.find(
      (it) =>
        it.children?.some((c) => isActivePath(pathname, c.href)) ||
        (it.children && isActivePath(pathname, it.href, it.exact)),
    )?.href ?? null;
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
            const active = isActivePath(pathname, item.href, item.exact);
            const childActive = item.children?.some((c) => isActivePath(pathname, c.href)) ?? false;
            const open = openHref === item.href;
            return (
              <SidebarMenuItem key={item.href}>
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
                      className="absolute top-2 right-1 flex size-7 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-card hover:text-foreground group-data-[collapsible=icon]:hidden"
                    >
                      <ChevronRight
                        className={cn("size-3.5 transition-transform duration-200", open && "rotate-90")}
                      />
                    </button>
                    {open ? (
                      <ul className="mt-1 mb-2 ml-4 space-y-0.5 rounded-2xl bg-card p-1.5 shadow-(--card-shadow) group-data-[collapsible=icon]:hidden">
                        {item.children.map((child) => {
                          const on = isActivePath(pathname, child.href);
                          return (
                            <li key={child.href}>
                              <Link
                                href={withContext(child.href)}
                                aria-current={on ? "page" : undefined}
                                className={cn(
                                  "flex h-8 items-center gap-2.5 rounded-full px-3 text-[13px] outline-none transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-ring",
                                  on
                                    ? "bg-muted font-medium text-foreground"
                                    : "text-muted-foreground hover:bg-muted/60 hover:text-foreground",
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
