"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Check, ChevronsUpDown, Plus } from "lucide-react";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command";
import { Kbd } from "@/components/ui/kbd";
import { cn } from "@/lib/utils";
import { initials } from "@/lib/format";
import { t } from "@/i18n/id";

export type SwitcherClient = {
  id: string;
  name: string;
  slug: string;
  industry?: string | null;
};

export function ClientSwitcher({
  clients,
  current,
  collapsed,
}: {
  clients: SwitcherClient[];
  current?: SwitcherClient;
  collapsed?: boolean;
}) {
  const [open, setOpen] = React.useState(false);
  const router = useRouter();

  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((o) => !o);
      }
    };
    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, []);

  const go = (href: string) => {
    setOpen(false);
    router.push(href);
  };

  return (
    <>
      <Button
        variant="outline"
        onClick={() => setOpen(true)}
        aria-label={t.nav.switchClient}
        className={cn(
          "h-10 w-full justify-start gap-2 bg-background px-2 text-left shadow-none",
          collapsed && "size-8 justify-center p-0",
        )}
      >
        <Avatar className="size-6 rounded-md">
          <AvatarFallback className="rounded-md bg-primary text-[10px] font-semibold text-primary-foreground">
            {current ? initials(current.name) : "W"}
          </AvatarFallback>
        </Avatar>
        {!collapsed ? (
          <>
            <span className="flex min-w-0 flex-1 flex-col leading-tight">
              <span className="truncate text-sm font-medium">
                {current?.name ?? t.nav.allClients}
              </span>
              <span className="truncate text-[11px] text-muted-foreground">
                {current?.industry ??
                  (current
                    ? t.nav.workspace
                    : `${clients.length} ${t.portfolio.clientsCount}`)}
              </span>
            </span>
            <ChevronsUpDown className="size-4 shrink-0 text-muted-foreground" />
          </>
        ) : null}
      </Button>

      <CommandDialog
        open={open}
        onOpenChange={setOpen}
        title={t.nav.switchClient}
        description={t.common.keyboardHint}
      >
        <Command>
          <CommandInput placeholder={t.nav.searchClient} />
          <CommandList>
            <CommandEmpty>{t.nav.noClient}</CommandEmpty>
            <CommandGroup heading={t.nav.clients}>
              {clients.map((c) => (
                <CommandItem
                  key={c.id}
                  value={`${c.name} ${c.slug}`}
                  onSelect={() => go(`/clients/${c.slug}`)}
                >
                  <Avatar className="size-6 rounded-md">
                    <AvatarFallback className="rounded-md text-[10px] font-semibold">
                      {initials(c.name)}
                    </AvatarFallback>
                  </Avatar>
                  <span className="flex-1 truncate">{c.name}</span>
                  {c.industry ? (
                    <span className="text-xs text-muted-foreground">
                      {c.industry}
                    </span>
                  ) : null}
                  {current?.id === c.id ? <Check className="size-4" /> : null}
                </CommandItem>
              ))}
            </CommandGroup>
            <CommandSeparator />
            <CommandGroup heading={t.nav.general}>
              <CommandItem onSelect={() => go("/")}>
                {t.nav.portfolio}
              </CommandItem>
              <CommandItem onSelect={() => go("/clients/new")}>
                <Plus className="size-4" /> {t.clients.new}
              </CommandItem>
            </CommandGroup>
          </CommandList>
          <div className="flex items-center gap-2 border-t px-3 py-2 text-[11px] text-muted-foreground">
            <Kbd>↑↓</Kbd> navigasi <Kbd>↵</Kbd> pilih <Kbd>esc</Kbd> tutup
          </div>
        </Command>
      </CommandDialog>
    </>
  );
}
