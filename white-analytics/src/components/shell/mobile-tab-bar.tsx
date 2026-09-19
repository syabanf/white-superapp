"use client";

import * as React from "react";
import Link from "next/link";
import { useParams, usePathname } from "next/navigation";
import { LayoutGrid } from "lucide-react";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { ClientSwitcher, type SwitcherClient } from "@/components/shell/client-switcher";
import { UserMenu, type ShellUser } from "@/components/shell/user-menu";
import { adminNav, clientNav, generalNav, isActivePath, type NavItem } from "@/components/shell/nav-config";
import { cn } from "@/lib/utils";
import { t } from "@/i18n/id";

/** Pages with their own sticky bottom action bar: the tab bar steps aside there. */
const HAS_OWN_BOTTOM_BAR = /^\/(setup|clients\/new)(\/|$)|\/publish\/(new|posts\/[^/]+)(\/|$)/;
const PRIMARY_SLOTS = 4;

/**
 * Phone navigation: a floating ink pill with the four most used destinations
 * and a "more" button. "More" opens a bottom sheet holding the whole menu,
 * the client switcher and the account menu, so the side drawer is not needed
 * on small screens.
 */
export function MobileTabBar({ clients, user }: { clients: SwitcherClient[]; user: ShellUser }) {
  const pathname = usePathname();
  const params = useParams<{ slug?: string }>();
  const slug = typeof params?.slug === "string" ? params.slug : undefined;
  const client = slug ? clients.find((c) => c.slug === slug) : undefined;
  const [open, setOpen] = React.useState(false);

  if (HAS_OWN_BOTTOM_BAR.test(pathname)) return null;

  const groups: { label: string; items: NavItem[] }[] = [
    ...(client ? [{ label: client.name, items: clientNav(client.slug) }] : []),
    { label: t.nav.general, items: generalNav },
    ...(user.role === "ADMIN" ? [{ label: t.nav.admin, items: adminNav }] : []),
  ];
  const primary = groups.flatMap((g) => g.items).slice(0, PRIMARY_SLOTS);
  const isOn = (item: NavItem) => isActivePath(pathname, item.href, item.exact);
  const moreActive = !primary.some(isOn);

  return (
    <nav
      aria-label={t.nav.modules}
      className="fixed inset-x-4 bottom-[max(0.75rem,env(safe-area-inset-bottom))] z-40 flex h-16 items-center justify-around rounded-full bg-primary px-3 text-primary-foreground shadow-[0_18px_40px_-16px_rgba(16,19,31,0.65)] md:hidden"
    >
      {primary.map((item) => (
        <Link key={item.href} href={item.href} aria-label={item.label} aria-current={isOn(item) ? "page" : undefined} className={tabClass(isOn(item))}>
          <item.icon className="size-5" />
        </Link>
      ))}

      <Sheet open={open} onOpenChange={setOpen}>
        <SheetTrigger asChild>
          <button type="button" aria-label={t.nav.more} className={tabClass(moreActive)}>
            <LayoutGrid className="size-5" />
          </button>
        </SheetTrigger>
        <SheetContent side="bottom" className="max-h-[85dvh] gap-0 overflow-y-auto rounded-t-[1.75rem] border-t-0 bg-background pb-[max(1rem,env(safe-area-inset-bottom))]">
          <SheetHeader className="pb-2">
            <span aria-hidden className="mx-auto mb-2 h-1 w-10 rounded-full bg-border" />
            <SheetTitle className="text-lg font-semibold tracking-[-0.02em]">{t.nav.more}</SheetTitle>
            <SheetDescription className="sr-only">{t.nav.modules}</SheetDescription>
          </SheetHeader>

          <div className="space-y-4 px-4">
            <ClientSwitcher clients={clients} current={client} />
            {groups.map((g) => (
              <section key={g.label}>
                <h3 className="label-mono mb-1.5 px-1 text-muted-foreground">{g.label}</h3>
                <ul className="rounded-2xl bg-card p-1.5 shadow-(--card-shadow)">
                  {g.items.flatMap((item) => [item, ...(item.children ?? []).map((c) => ({ ...c, child: true as const }))]).map((entry) => {
                    const child = "child" in entry;
                    const on = child ? isActivePath(pathname, entry.href) : isOn(entry as NavItem);
                    const Icon = entry.icon;
                    return (
                      <li key={entry.href}>
                        <Link
                          href={entry.href}
                          onClick={() => setOpen(false)}
                          aria-current={on ? "page" : undefined}
                          className={cn(
                            "flex h-11 items-center gap-3 rounded-full px-3 text-sm transition-colors",
                            child && "pl-12 text-[13px]",
                            on ? "bg-muted font-medium text-foreground" : "text-muted-foreground",
                          )}
                        >
                          {!child && Icon ? <Icon className={cn("size-4.5", on && "text-brand")} /> : null}
                          <span className="truncate">{entry.label}</span>
                        </Link>
                      </li>
                    );
                  })}
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

function tabClass(active: boolean) {
  return cn(
    "flex size-12 items-center justify-center rounded-full outline-none transition-[background-color,color,transform] duration-200 focus-visible:ring-2 focus-visible:ring-ring active:scale-95",
    active ? "bg-brand text-white shadow-[0_8px_20px_-6px_var(--brand)]" : "text-primary-foreground/70 hover:text-primary-foreground",
  );
}
