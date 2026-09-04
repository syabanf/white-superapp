"use client";

import Link from "next/link";
import { useParams, usePathname } from "next/navigation";
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
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarRail,
  useSidebar,
} from "@/components/ui/sidebar";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Button } from "@/components/ui/button";
import { WhiteLogo } from "@/components/brand/logo";
import { ClientSwitcher, type SwitcherClient } from "@/components/shell/client-switcher";
import { UserMenu, type ShellUser } from "@/components/shell/user-menu";
import { adminNav, clientNav, generalNav, isActivePath, type NavItem } from "@/components/shell/nav-config";
import { cn } from "@/lib/utils";
import { t } from "@/i18n/id";

export function AppSidebar({ clients, user }: { clients: SwitcherClient[]; user: ShellUser }) {
  const pathname = usePathname();
  const params = useParams<{ slug?: string }>();
  const slug = typeof params?.slug === "string" ? params.slug : undefined;
  const currentClient = slug ? clients.find((c) => c.slug === slug) : undefined;
  const { state } = useSidebar();
  const collapsed = state === "collapsed";

  return (
    <Sidebar collapsible="icon" variant="sidebar" className="border-r">
      <SidebarHeader className="gap-4 px-4 pt-5">
        <Link
          href="/"
          className="flex h-8 items-center gap-2 outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <WhiteLogo variant={collapsed ? "mark" : "full"} size={collapsed ? 22 : 26} />
          {!collapsed ? (
            <span className="label-mono ml-auto text-muted-foreground">Analytics</span>
          ) : null}
        </Link>
        <ClientSwitcher clients={clients} current={currentClient} collapsed={collapsed} />
      </SidebarHeader>

      <SidebarContent className="scrollbar-thin px-1">
        <NavGroup label={t.nav.general} items={generalNav} pathname={pathname} />
        {currentClient ? (
          <NavGroup label={currentClient.name} items={clientNav(currentClient.slug)} pathname={pathname} />
        ) : null}
        {user.role === "ADMIN" ? <NavGroup label={t.nav.admin} items={adminNav} pathname={pathname} /> : null}
      </SidebarContent>

      <SidebarFooter className="gap-3 p-3">
        {!collapsed ? (
          <Button asChild className="h-10 w-full justify-center">
            {currentClient ? (
              <Link href={`/clients/${currentClient.slug}/reports`}>
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

/** Active marker: a blue dot, the withwhite.id nav rhythm — rows split by hairlines. */
function Dot({ active }: { active: boolean }) {
  return (
    <span
      aria-hidden
      className={cn(
        "size-1.5 shrink-0 rounded-full transition-all duration-200 group-data-[collapsible=icon]:hidden",
        active ? "scale-100 bg-brand" : "scale-50 bg-muted-foreground/30",
      )}
    />
  );
}

function NavGroup({ label, items, pathname }: { label: string; items: NavItem[]; pathname: string }) {
  return (
    <SidebarGroup className="px-2 py-1.5">
      <SidebarGroupLabel className="label-mono h-auto truncate px-2 pb-1.5 text-muted-foreground">
        {label}
      </SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu className="gap-0">
          {items.map((item, i) => {
            const active = isActivePath(pathname, item.href, item.exact);
            const last = i === items.length - 1;
            const rowBorder = !last ? "border-b border-sidebar-border/60" : "";

            if (item.children?.length) {
              const childActive = item.children.some((c) => isActivePath(pathname, c.href));
              return (
                <Collapsible
                  key={item.href}
                  asChild
                  defaultOpen={active || childActive}
                  className="group/collapsible"
                >
                  <SidebarMenuItem className={rowBorder}>
                    <SidebarMenuButton
                      asChild
                      tooltip={item.label}
                      isActive={active && !childActive}
                      className="h-9 gap-2.5 rounded-md px-2 font-normal transition-colors duration-150 data-[active=true]:bg-transparent data-[active=true]:font-medium data-[active=true]:text-brand-ink"
                    >
                      <Link href={item.href}>
                        <Dot active={active || childActive} />
                        <item.icon className="hidden text-muted-foreground group-data-[collapsible=icon]:block" />
                        <span>{item.label}</span>
                      </Link>
                    </SidebarMenuButton>
                    <CollapsibleTrigger asChild>
                      <button
                        type="button"
                        aria-label={`${t.nav.expand} ${item.label}`}
                        className="absolute top-1.5 right-1 flex size-6 items-center justify-center rounded-md text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground group-data-[collapsible=icon]:hidden"
                      >
                        <ChevronRight className="size-3.5 transition-transform group-data-[state=open]/collapsible:rotate-90" />
                      </button>
                    </CollapsibleTrigger>
                    <CollapsibleContent>
                      <SidebarMenuSub className="mr-0 ml-3.5 gap-0 border-l border-sidebar-border/70 px-0">
                        {item.children.map((child) => {
                          const ca = isActivePath(pathname, child.href);
                          return (
                            <SidebarMenuSubItem key={child.href}>
                              <SidebarMenuSubButton
                                asChild
                                isActive={ca}
                                className="h-8 gap-2 rounded-none border-l-2 border-transparent pl-3 text-[13px] font-normal data-[active=true]:border-brand data-[active=true]:bg-transparent data-[active=true]:font-medium data-[active=true]:text-brand-ink"
                              >
                                <Link href={child.href}>
                                  <span>{child.label}</span>
                                </Link>
                              </SidebarMenuSubButton>
                            </SidebarMenuSubItem>
                          );
                        })}
                      </SidebarMenuSub>
                    </CollapsibleContent>
                  </SidebarMenuItem>
                </Collapsible>
              );
            }

            return (
              <SidebarMenuItem key={item.href} className={rowBorder}>
                <SidebarMenuButton
                  asChild
                  tooltip={item.label}
                  isActive={active}
                  className="h-9 gap-2.5 rounded-md px-2 font-normal transition-colors duration-150 data-[active=true]:bg-transparent data-[active=true]:font-medium data-[active=true]:text-brand-ink"
                >
                  <Link href={item.href}>
                    <Dot active={active} />
                    <item.icon className="hidden text-muted-foreground group-data-[collapsible=icon]:block" />
                    <span>{item.label}</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            );
          })}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
