"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Check, ChevronsUpDown, Layers, Megaphone } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { workspaceOf, type Workspace } from "@/components/shell/nav-config";
import { cn } from "@/lib/utils";
import { t } from "@/i18n/id";

const WORKSPACES: { key: Workspace; href: string; label: string; hint: string; icon: typeof Megaphone }[] = [
  {
    key: "marketing",
    href: "/",
    label: t.workspace.marketing,
    hint: t.workspace.marketingHint,
    icon: Megaphone,
  },
  { key: "erp", href: "/erp", label: t.workspace.erp, hint: t.workspace.erpHint, icon: Layers },
];

/**
 * Separates the two products: Digital Marketing Platform and ERP. Each has
 * its own home and menu; the chip colour follows the workspace accent.
 */
export function WorkspaceSwitcher({ collapsed, showErp }: { collapsed?: boolean; showErp: boolean }) {
  const pathname = usePathname();
  const current = WORKSPACES.find((w) => w.key === workspaceOf(pathname)) ?? WORKSPACES[0];
  const options = showErp ? WORKSPACES : WORKSPACES.filter((w) => w.key === "marketing");
  const Icon = current.icon;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label={t.workspace.switch}
        className={cn(
          "flex w-full items-center gap-2.5 rounded-full bg-muted/70 p-1.5 text-left outline-none transition-colors hover:bg-muted focus-visible:ring-2 focus-visible:ring-ring",
          collapsed && "size-11 justify-center p-0",
        )}
      >
        <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-[linear-gradient(135deg,var(--brand),color-mix(in_oklab,var(--brand)_60%,var(--logo-navy)))] text-white shadow-[0_6px_14px_-6px_var(--brand)]">
          <Icon className="size-4" />
        </span>
        {!collapsed ? (
          <>
            <span className="min-w-0 flex-1">
              <span className="block text-[11px] leading-tight text-muted-foreground">
                {t.workspace.label}
              </span>
              <span className="block truncate text-sm font-medium leading-tight">{current.label}</span>
            </span>
            <ChevronsUpDown className="mr-1.5 size-3.5 shrink-0 text-muted-foreground" />
          </>
        ) : null}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-72 rounded-2xl p-1.5">
        <DropdownMenuLabel className="text-xs font-normal text-muted-foreground">
          {t.workspace.switch}
        </DropdownMenuLabel>
        {options.map((w) => {
          const WIcon = w.icon;
          const on = w.key === current.key;
          return (
            <DropdownMenuItem key={w.key} asChild className="gap-3 rounded-xl p-2">
              <Link href={w.href}>
                <span
                  className={cn(
                    "flex size-8 shrink-0 items-center justify-center rounded-full",
                    w.key === "erp"
                      ? "bg-[var(--erp-accent)] text-white"
                      : "bg-[var(--marketing-accent)] text-white",
                  )}
                >
                  <WIcon className="size-4" />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block text-sm font-medium">{w.label}</span>
                  <span className="block truncate text-xs text-muted-foreground">{w.hint}</span>
                </span>
                {on ? <Check className="size-4 text-brand" /> : null}
              </Link>
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
