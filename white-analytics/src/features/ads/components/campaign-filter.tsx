"use client";

import { useTransition } from "react";
import { parseAsString, useQueryState } from "nuqs";
import { Loader2, Megaphone } from "lucide-react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { s } from "@/features/ads/strings";

const ALL = "all";

/**
 * Filter kampanye (?campaign=<id>) — seluruh halaman menghormati filter ini
 * karena nilai dibaca server-side oleh getAdsDashboard.
 */
export function CampaignFilter({ options, className }: { options: { id: string; name: string }[]; className?: string }) {
  const [isPending, startTransition] = useTransition();
  const [campaign, setCampaign] = useQueryState(
    "campaign",
    parseAsString.withOptions({ shallow: false, history: "replace", startTransition }),
  );
  const value = campaign && options.some((o) => o.id === campaign) ? campaign : ALL;

  return (
    <div className={className}>
      <Select value={value} onValueChange={(v) => void setCampaign(v === ALL ? null : v)}>
        <SelectTrigger size="sm" className="w-full max-w-xs bg-background" aria-label={s.filterCampaign}>
          {isPending ? (
            <Loader2 className="size-4 shrink-0 animate-spin text-muted-foreground" />
          ) : (
            <Megaphone className="size-4 shrink-0 text-muted-foreground" />
          )}
          <SelectValue placeholder={s.allCampaigns} />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>{s.allCampaigns}</SelectItem>
          {options.map((o) => (
            <SelectItem key={o.id} value={o.id}>
              {o.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
