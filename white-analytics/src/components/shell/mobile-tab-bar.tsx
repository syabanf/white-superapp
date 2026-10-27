"use client";

import * as React from "react";
import Link from "next/link";
import { useParams, usePathname } from "next/navigation";
import { ChevronDown, LayoutGrid } from "lucide-react";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
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

/** Pages with their own sticky bottom action bar: the tab bar steps aside there. */
const HAS_OWN_BOTTOM_BAR = /^\/(setup|clients\/new)(\/|$)|\/publish\/(new|posts\/[^/]+)(\/|$)/;
const PRIMARY_SLOTS = 4;

/**
 * Phone navigation: a floating navy pill with the four most used destinations
 * and a "more" button. "More" opens a bottom sheet holding the whole menu,
 * the client switcher and the account menu, so the side drawer is not needed
 * on small screens.
 */
export function MobileTabBar({
  clients,
  user,
  erpAccess,
}: {
  clients: SwitcherClient[];
  user: ShellUser;
  erpAccess: ErpAccess | null;
}) {
  const pathname = usePathname();
  const params = useParams<{ slug?: string }>();
  const slug = typeof params?.slug === "string" ? params.slug : undefined;
  const client = slug ? clients.find((c) => c.slug === slug) : undefined;
  const [open, setOpen] = React.useState(false);

  if (HAS_OWN_BOTTOM_BAR.test(pathname)) return null;

  const inErp = workspaceOf(pathname) === "erp";
  const erpGroups = visibleErpNav(erpAccess);
  const groups: { label: string; items: NavItem[] }[] = inErp
    ? erpGroups
    : [
        ...(client ? [{ label: client.name, items: clientNav(client.slug) }] : []),
        { label: t.nav.general, items: generalNav },
        ...(user.role === "ADMIN" ? [{ label: t.nav.admin, items: adminNav }] : []),
      ];
  const primary = groups.flatMap((g) => g.items).slice(0, PRIMARY_SLOTS);
  const current = activeNavHref(pathname, groups.flatMap((g) => g.items));
  const isOn = (item: NavItem) => item.href === current || (item.children?.some((c) => c.href === current) ?? false);
  const moreActive = !primary.some(isOn);

  return (
    <nav
      aria-label={t.nav.modules}
      className="fixed inset-x-4 bottom-[max(0.75rem,env(safe-area-inset-bottom))] z-40 flex h-16 items-center justify-around rounded-full bg-[linear-gradient(120deg,rgba(43,34,99,0.96),rgba(84,36,104,0.96))] px-2.5 text-white shadow-[0_20px_40px_-14px_rgba(43,34,99,0.7)] dark:bg-[linear-gradient(120deg,rgba(62,46,130,0.95),rgba(112,40,120,0.95))] ring-1 ring-white/10 md:hidden"
    >
      {primary.map((item) => (
        <Link
          key={item.href}
          href={item.href}
          aria-label={item.label}
          aria-current={isOn(item) ? "page" : undefined}
          className={tabClass(isOn(item))}
        >
          <item.icon className="size-5" />
        </Link>
      ))}

      <Sheet open={open} onOpenChange={setOpen}>
        <SheetTrigger asChild>
          <button type="button" aria-label={t.nav.more} className={tabClass(moreActive)}>
            <LayoutGrid className="size-5" />
          </button>
        </SheetTrigger>
        <SheetContent
          side="bottom"
          className="max-h-[85dvh] gap-0 overflow-y-auto pb-[max(1rem,env(safe-area-inset-bottom))]"
        >
          <SheetHeader className="pb-2">
            <span aria-hidden className="mx-auto mb-2 h-1 w-10 rounded-full bg-border" />
            <SheetTitle className="text-lg font-semibold tracking-[-0.02em]">{t.nav.more}</SheetTitle>
            <SheetDescription className="sr-only">{t.nav.modules}</SheetDescription>
          </SheetHeader>

          <div className="space-y-4 px-4">
            <WorkspaceSwitcher showErp={erpGroups.length > 1} />
            {!inErp ? <ClientSwitcher clients={clients} current={client} /> : null}
            {groups.map((g) => (
              <section key={g.label}>
                <h3 className="label-mono mb-1.5 px-1 text-muted-foreground">{g.label}</h3>
                <ul className="rounded-3xl bg-card p-1.5 shadow-(--card-shadow)">
                  {g.items.map((item) => <MobileMenuItem key={`${item.href}:${pathname}`} item={item} current={current} onNavigate={() => setOpen(false)} />)}
                </ul>
              </section>
            ))}
            <UserMenu user={user} />
          </div>
        </SheetContent>
      </Sheet>
    </nav>
  );
}

function MobileMenuItem({ item, current, onNavigate }: { item: NavItem; current: string | null; onNavigate: () => void }) {
  const on = item.href === current;
  const childOn = item.children?.some((c) => c.href === current) ?? false;
  const [expanded, setExpanded] = React.useState(on || childOn);
  return <li>
    <div className={cn("flex items-center rounded-2xl", (on || childOn) && "bg-[color-mix(in_oklab,var(--brand)_9%,transparent)]")}>
      <Link
        href={item.href}
        onClick={onNavigate}
        aria-current={on && !childOn ? "page" : undefined}
        className={cn("flex h-11 min-w-0 flex-1 items-center gap-3 rounded-full px-3 text-sm", on || childOn ? "font-medium text-foreground" : "text-muted-foreground")}
      >
        <item.icon className={cn("size-4.5 shrink-0", (on || childOn) && "text-brand")} />
        <span className="truncate">{item.label}</span>
      </Link>
      {item.children?.length ? <button
        type="button"
        aria-label={`${expanded ? "Ciutkan" : "Perluas"} ${item.label}`}
        aria-expanded={expanded}
        onClick={() => setExpanded(!expanded)}
        className="mr-1 flex size-10 shrink-0 items-center justify-center rounded-full text-muted-foreground hover:bg-background"
      ><ChevronDown className={cn("size-4 transition-transform", !expanded && "-rotate-90")} /></button> : null}
    </div>
    {expanded && item.children?.length ? <ul className="mb-2 ml-10 border-l pl-2">
      {item.children.map((child) => {
        const active = child.href === current;
        return <li key={child.href}><Link
          href={child.href}
          onClick={onNavigate}
          aria-current={active ? "page" : undefined}
          className={cn("flex min-h-10 items-center rounded-full px-3 text-[13px]", active ? "bg-muted font-medium text-foreground" : "text-muted-foreground")}
        >{child.label}</Link></li>;
      })}
    </ul> : null}
  </li>;
}

function tabClass(active: boolean) {
  return cn(
    "flex size-12 items-center justify-center rounded-full outline-none transition-[background-color,color,transform] duration-200 focus-visible:ring-2 focus-visible:ring-ring active:scale-95",
    active
      ? "bg-white text-brand shadow-[0_8px_20px_-8px_rgba(255,255,255,0.6)]"
      : "text-white/70 hover:text-white",
  );
}
