"use client";

import * as React from "react";
import { EmptyState } from "@/components/dashboard/empty-state";
import { Toolbar, ToolbarActions } from "@/components/dashboard/layout";
import Link from "next/link";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { ArrowUpRight, CircleAlert, Clock, Megaphone, Search, Share2 } from "lucide-react";
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
import { cn } from "@/lib/utils";
import { t } from "@/i18n/id";
import { tc } from "@/features/clients/strings";

export type PortfolioCardData = {
  id: string;
  name: string;
  slug: string;
  industry: string | null;
  /** Klien contoh: selalu data demo. */
  isDemo: boolean;
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
  coverage: { social: boolean; seo: boolean; ads: boolean };
  /** pre-formatted relative label (computed server-side to avoid hydration drift) */
  lastSyncLabel: string | null;
  contextQuery: string;
  attention: { label: string; href: string; kind: "warning" | "critical" }[];
};

const SORT_KEYS = ["name", "followers", "clicks", "spend", "health"] as const;
type SortKey = (typeof SORT_KEYS)[number];

const SORT_OPTIONS: { key: SortKey; label: string }[] = [
  { key: "name", label: tc.portfolio.sortName },
  { key: "followers", label: tc.portfolio.sortFollowers },
  { key: "clicks", label: tc.portfolio.sortClicks },
  { key: "spend", label: tc.portfolio.sortSpend },
  { key: "health", label: tc.portfolio.sortHealth },
];

export function PortfolioGrid({ cards }: { cards: PortfolioCardData[] }) {
  const [query, setQuery] = React.useState("");
  // in the URL so the portfolio hero stats can open the grid sorted by their metric
  const [sort, setSort] = useQueryState(
    "sort",
    parseAsStringLiteral(SORT_KEYS).withDefault("name").withOptions({ history: "replace", scroll: false }),
  );

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
    <div id="clients" className="flex scroll-mt-20 flex-col gap-(--gap-inner)">
      <Toolbar>
        <div className="relative max-sm:w-full">
          <Search className="pointer-events-none absolute top-1/2 left-3.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={tc.portfolio.search}
            className="w-full pl-9 sm:w-64"
            aria-label={tc.portfolio.search}
          />
        </div>
        <ToolbarActions>
          <span className="text-xs text-muted-foreground">{tc.portfolio.sort}</span>
          <Select value={sort} onValueChange={(v) => void setSort(v as SortKey)}>
            <SelectTrigger className="w-40" aria-label={tc.portfolio.sort}>
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
        </ToolbarActions>
      </Toolbar>

      {visible.length === 0 ? (
        <EmptyState compact state="zero" title={tc.portfolio.noMatch} />
      ) : (
        <div className="grid gap-(--gap) sm:grid-cols-2 xl:grid-cols-3">
          {visible.map((c) => (
            <ClientCard key={c.id} data={c} />
          ))}
        </div>
      )}
    </div>
  );
}

function ClientCard({ data }: { data: PortfolioCardData }) {
  const href = (path = "") => {
    const base = `/clients/${data.slug}${path ? `/${path}` : ""}`;
    if (!data.contextQuery) return base;
    const separator = base.includes("?") ? "&" : "?";
    return `${base}${separator}${data.contextQuery}`;
  };
  const activeModules: { key: string; label: string; icon: typeof Share2 }[] = [];
  if (data.modules.social)
    activeModules.push({ key: "social", label: tc.portfolio.moduleSocial, icon: Share2 });
  if (data.modules.seo) activeModules.push({ key: "seo", label: tc.portfolio.moduleSeo, icon: Search });
  if (data.modules.ads) activeModules.push({ key: "ads", label: tc.portfolio.moduleAds, icon: Megaphone });

  return (
    <Card className="lift flex flex-col gap-4 p-(--card-pad) hover:ring-brand/25">
      <div className="flex items-start justify-between gap-3">
        <Link href={href()} className="group flex min-w-0 items-center gap-3">
          <Avatar className="size-11 rounded-full">
            {data.logoUrl ? <AvatarImage src={data.logoUrl} alt={data.name} /> : null}
            <AvatarFallback className="rounded-full bg-accent text-sm font-semibold text-accent-foreground">
              {initials(data.name)}
            </AvatarFallback>
          </Avatar>
          <span className="min-w-0">
            <span className="line-clamp-2 text-[15px] leading-snug font-semibold tracking-[-0.015em] group-hover:underline">
              {data.name}
            </span>
            <span className="flex items-center gap-1.5 truncate text-xs text-muted-foreground">
              {data.isDemo ? <Badge variant="outline" className="h-4 px-1.5 text-[10px]">{tc.settings.demoBadge}</Badge> : null}
              <span className="truncate">{data.industry ?? "–"}</span>
            </span>
          </span>
        </Link>
        {data.health != null ? (
          <span title={tc.portfolio.scoreLabel} className="shrink-0">
            <StatusBadge kind={ratingToKind(rateScore(data.health))}>{formatNumber(data.health)}</StatusBadge>
          </span>
        ) : null}
      </div>

      <div className="space-y-0.5 rounded-2xl bg-muted/70 p-1.5">
        <MiniMetric label={tc.portfolio.metricFollowers} value={formatCompact(data.followers)} delta={data.followersDelta} href={href("social")} />
        <MiniMetric
          label={tc.portfolio.metricClicks}
          value={data.coverage.seo ? formatCompact(data.clicks) : "–"}
          delta={data.coverage.seo ? data.clicksDelta : null}
          href={href("seo")}
        />
        <MiniMetric
          label={tc.portfolio.metricSpend}
          value={data.coverage.ads ? formatCurrency(data.spend, data.currency, { compact: true }) : "–"}
          delta={data.coverage.ads ? data.spendDelta : null}
          href={href("ads")}
        />
      </div>
      {!data.coverage.social || !data.coverage.seo || !data.coverage.ads ? <p className="text-xs text-muted-foreground">– berarti belum ada data pada periode terpilih.</p> : null}

      {activeModules.length > 0 ? (
        <div className="flex flex-wrap items-center gap-1.5" aria-label={t.portfolio.modulesActive}>
          {activeModules.map((m) => (
            <span key={m.key} className="inline-flex h-6 items-center gap-1 rounded-full bg-muted px-2.5 text-xs text-muted-foreground">
              <m.icon className="size-3" /> {m.label}
            </span>
          ))}
        </div>
      ) : null}

      {data.attention.length > 0 ? (
        <div className="space-y-1.5 border-t pt-3" aria-label="Perlu perhatian">
          {data.attention.map((item) => (
            <Link
              key={`${item.href}-${item.label}`}
              href={href(item.href)}
              className="flex items-center gap-2 rounded-md px-1 py-1 text-xs text-muted-foreground hover:bg-muted hover:text-foreground"
            >
              <CircleAlert
                className={cn("size-3.5", item.kind === "critical" ? "text-negative" : "text-warning")}
              />
              <span className="flex-1">{item.label}</span>
              <ArrowUpRight className="size-3.5" />
            </Link>
          ))}
        </div>
      ) : null}

      <div className="mt-auto flex items-center justify-between gap-2 border-t pt-3">
        <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
          <Clock className="size-3.5" />
          {tc.portfolio.lastSync}: {data.lastSyncLabel ?? t.common.neverSynced}
        </span>
        <Button asChild size="xs" variant="outline" className="group/nudge">
          <Link href={href()}>
            Buka rincian <ArrowUpRight className="nudge size-3.5" />
          </Link>
        </Button>
      </div>
    </Card>
  );
}

/** One metric row: label left, value (and delta when comparing) right. */
function MiniMetric({ label, value, delta, href }: { label: string; value: string; delta: Delta | null; href: string }) {
  return (
    <Link
      href={href}
      className="flex items-center justify-between gap-3 rounded-xl px-3 py-2 outline-none transition-colors hover:bg-background/70 focus-visible:ring-2 focus-visible:ring-ring"
    >
      <span className="label-mono truncate text-muted-foreground">{label}</span>
      <span className="flex shrink-0 items-center gap-2">
        <span className="text-[15px] font-semibold tracking-[-0.02em] tabular">{value}</span>
        {delta ? <DeltaBadge delta={delta} /> : null}
      </span>
    </Link>
  );
}
