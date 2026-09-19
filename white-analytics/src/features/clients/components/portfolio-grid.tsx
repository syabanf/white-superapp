"use client";

import * as React from "react";
import Link from "next/link";
import { ArrowUpRight, Clock, Megaphone, Search, Share2 } from "lucide-react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { DeltaBadge } from "@/components/dashboard/delta-badge";
import { StatusBadge, ratingToKind } from "@/components/dashboard/status-badge";
import { formatCompact, formatCurrency, formatNumber, initials } from "@/lib/format";
import { rateScore, type Delta } from "@/lib/metrics";
import { t } from "@/i18n/id";
import { tc } from "@/features/clients/strings";

export type PortfolioCardData = {
  id: string;
  name: string;
  slug: string;
  industry: string | null;
  logoUrl: string | null;
  currency: string;
  followers: number;
  followersDelta: Delta | null;
  clicks: number;
  clicksDelta: Delta | null;
  spend: number;
  spendDelta: Delta | null;
  health: number | null;
  modules: { social: boolean; seo: boolean; ads: boolean };
  /** pre-formatted relative label (computed server-side to avoid hydration drift) */
  lastSyncLabel: string | null;
};

type SortKey = "name" | "followers" | "clicks" | "spend" | "health";

const SORT_OPTIONS: { key: SortKey; label: string }[] = [
  { key: "name", label: tc.portfolio.sortName },
  { key: "followers", label: tc.portfolio.sortFollowers },
  { key: "clicks", label: tc.portfolio.sortClicks },
  { key: "spend", label: tc.portfolio.sortSpend },
  { key: "health", label: tc.portfolio.sortHealth },
];

export function PortfolioGrid({ cards }: { cards: PortfolioCardData[] }) {
  const [query, setQuery] = React.useState("");
  const [sort, setSort] = React.useState<SortKey>("name");

  const visible = React.useMemo(() => {
    const q = query.trim().toLowerCase();
    const filtered = q
      ? cards.filter(
          (c) =>
            c.name.toLowerCase().includes(q) ||
            (c.industry ?? "").toLowerCase().includes(q) ||
            c.slug.includes(q),
        )
      : cards;
    const sorted = [...filtered];
    sorted.sort((a, b) => {
      switch (sort) {
        case "followers":
          return b.followers - a.followers;
        case "clicks":
          return b.clicks - a.clicks;
        case "spend":
          return b.spend - a.spend;
        case "health":
          return (b.health ?? -1) - (a.health ?? -1);
        default:
          return a.name.localeCompare(b.name, "id");
      }
    });
    return sorted;
  }, [cards, query, sort]);

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={tc.portfolio.search}
            className="h-8 w-60 pl-8 text-sm"
            aria-label={tc.portfolio.search}
          />
        </div>
        <div className="ml-auto flex items-center gap-2">
          <span className="text-xs text-muted-foreground">{tc.portfolio.sort}</span>
          <Select value={sort} onValueChange={(v) => setSort(v as SortKey)}>
            <SelectTrigger size="sm" className="w-40" aria-label={tc.portfolio.sort}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {SORT_OPTIONS.map((o) => (
                <SelectItem key={o.key} value={o.key}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {visible.length === 0 ? (
        <p className="rounded-xl border border-dashed bg-card/50 py-10 text-center text-sm text-muted-foreground">
          {tc.portfolio.noMatch}
        </p>
      ) : (
        <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
          {visible.map((c) => (
            <ClientCard key={c.id} data={c} />
          ))}
        </div>
      )}
    </div>
  );
}

function ClientCard({ data }: { data: PortfolioCardData }) {
  const activeModules: { key: string; label: string; icon: typeof Share2 }[] = [];
  if (data.modules.social)
    activeModules.push({ key: "social", label: tc.portfolio.moduleSocial, icon: Share2 });
  if (data.modules.seo) activeModules.push({ key: "seo", label: tc.portfolio.moduleSeo, icon: Search });
  if (data.modules.ads) activeModules.push({ key: "ads", label: tc.portfolio.moduleAds, icon: Megaphone });

  return (
    <Card className="lift flex flex-col gap-4 p-6 hover:ring-brand/25">
      <div className="flex items-start justify-between gap-3">
        <Link href={`/clients/${data.slug}`} className="group flex min-w-0 items-center gap-3">
          <Avatar className="size-10 rounded-lg">
            {data.logoUrl ? <AvatarImage src={data.logoUrl} alt={data.name} /> : null}
            <AvatarFallback className="rounded-lg bg-primary/10 text-sm font-semibold text-primary">
              {initials(data.name)}
            </AvatarFallback>
          </Avatar>
          <span className="min-w-0">
            <span className="block truncate text-sm font-semibold group-hover:underline">{data.name}</span>
            <span className="block truncate text-xs text-muted-foreground">{data.industry ?? "–"}</span>
          </span>
        </Link>
        {data.health != null ? (
          <StatusBadge kind={ratingToKind(rateScore(data.health))}>
            {tc.portfolio.scoreLabel} {formatNumber(data.health)}
          </StatusBadge>
        ) : (
          <StatusBadge kind="neutral">{tc.portfolio.noAudit}</StatusBadge>
        )}
      </div>

      <div className="divide-y rounded-lg border bg-muted/25">
        <MiniMetric
          label={tc.portfolio.metricFollowers}
          value={formatCompact(data.followers)}
          delta={data.followersDelta}
        />
        <MiniMetric
          label={tc.portfolio.metricClicks}
          value={formatCompact(data.clicks)}
          delta={data.clicksDelta}
        />
        <MiniMetric
          label={tc.portfolio.metricSpend}
          value={formatCurrency(data.spend, data.currency, { compact: true })}
          delta={data.spendDelta}
        />
      </div>

      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-xs text-muted-foreground">{t.portfolio.modulesActive}:</span>
        {activeModules.length === 0 ? (
          <span className="text-xs text-muted-foreground">{tc.portfolio.noModules}</span>
        ) : (
          activeModules.map((m) => (
            <Badge key={m.key} variant="secondary" className="gap-1 px-1.5 py-0 text-xs font-medium">
              <m.icon className="size-3" /> {m.label}
            </Badge>
          ))
        )}
      </div>

      <div className="mt-auto flex items-center justify-between gap-2 border-t pt-3">
        <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
          <Clock className="size-3.5" />
          {tc.portfolio.lastSync}: {data.lastSyncLabel ?? t.common.neverSynced}
        </span>
        <Button asChild size="xs" variant="outline" className="group/nudge">
          <Link href={`/clients/${data.slug}`}>
            {t.common.open} <ArrowUpRight className="nudge size-3.5" />
          </Link>
        </Button>
      </div>
    </Card>
  );
}

/** One spec row: label left, value + delta right. Stays legible at any card width. */
function MiniMetric({ label, value, delta }: { label: string; value: string; delta: Delta | null }) {
  return (
    <div className="flex items-center justify-between gap-2 px-2.5 py-1.5">
      <span className="label-mono truncate text-xs text-muted-foreground">{label}</span>
      <span className="flex shrink-0 items-center gap-1.5">
        <span className="text-[13px] font-bold tracking-[-0.02em] tabular">{value}</span>
        {delta ? <DeltaBadge delta={delta} /> : null}
      </span>
    </div>
  );
}
