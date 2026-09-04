"use client";

import Link from "next/link";
import { Checkbox } from "@/components/ui/checkbox";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { PlatformIcon, platformLabel } from "@/features/social/components/platform-icon";
import { PLATFORMS, type Platform } from "@/features/publishing/lib";
import type { OwnAccount } from "@/features/publishing/queries";
import { p } from "@/features/publishing/strings";
import { t } from "@/i18n/id";
import { initials } from "@/lib/format";
import { cn } from "@/lib/utils";

export function TargetPicker({
  accounts,
  selected,
  onChange,
  settingsHref,
  disabled,
}: {
  accounts: OwnAccount[];
  selected: string[];
  onChange: (ids: string[]) => void;
  settingsHref: string;
  disabled?: boolean;
}) {
  if (accounts.length === 0) {
    return (
      <p className="rounded-lg border border-dashed px-3 py-4 text-sm text-muted-foreground">
        {p.noAccounts}{" "}
        <Link href={settingsHref} className="text-brand underline-offset-4 hover:underline">
          {t.common.settings}
        </Link>
      </p>
    );
  }
  const groups = PLATFORMS.map((pl) => ({ platform: pl, items: accounts.filter((a) => a.platform === pl) })).filter((g) => g.items.length > 0);
  const toggle = (id: string, on: boolean) => onChange(on ? Array.from(new Set([...selected, id])) : selected.filter((x) => x !== id));

  return (
    <div className="space-y-3">
      {groups.map((g) => (
        <div key={g.platform}>
          <p className="label-mono mb-1.5 flex items-center gap-1.5 text-muted-foreground">
            <PlatformIcon platform={g.platform} className="size-3" mono /> {platformLabel(g.platform)}
          </p>
          <ul className="grid gap-1.5 sm:grid-cols-2">
            {g.items.map((a) => {
              const on = selected.includes(a.id);
              return (
                <li key={a.id}>
                  <label
                    className={cn(
                      "flex cursor-pointer items-center gap-2.5 rounded-lg border px-2.5 py-2 transition-colors duration-150 hover:bg-muted/50",
                      on && "border-brand/40 bg-brand/5",
                      disabled && "pointer-events-none opacity-60",
                    )}
                  >
                    <Checkbox checked={on} onCheckedChange={(v) => toggle(a.id, v === true)} disabled={disabled} aria-label={`@${a.username}`} />
                    <Avatar className="size-7">
                      <AvatarImage src={a.avatarUrl ?? undefined} alt="" />
                      <AvatarFallback className="text-[10px]">{initials(a.displayName || a.username)}</AvatarFallback>
                    </Avatar>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-[13px] font-medium leading-tight">{a.displayName}</span>
                      <span className="block truncate text-xs text-muted-foreground">@{a.username}</span>
                    </span>
                    <PlatformIcon platform={a.platform as Platform} className="size-4" />
                  </label>
                </li>
              );
            })}
          </ul>
        </div>
      ))}
      <p className="text-xs text-muted-foreground">{p.targetsHint}</p>
    </div>
  );
}
