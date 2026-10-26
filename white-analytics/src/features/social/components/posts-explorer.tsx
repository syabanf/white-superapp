"use client";

import * as React from "react";
import { useQueryStates } from "nuqs";
import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { PostsTable } from "@/features/social/components/posts-table";
import { filterPosts, POST_SORTS, postFilterParsers, type PostSort } from "@/features/social/post-filters";
import type { SocialPostRow } from "@/features/social/queries";
import { s } from "@/features/social/strings";
import { formatNumber } from "@/lib/format";
import { t } from "@/i18n/id";

const ALL = "all";
const HOURS = Array.from({ length: 24 }, (_, h) => h);

/**
 * Postingan periode ini dengan filter tipe, hari, dan jam di URL
 * (`?type=Reels&day=0&hour=19&sort=er#posts`), tujuan drill dari kartu dan grafik Social.
 */
export function PostsExplorer({ rows, exportName }: { rows: SocialPostRow[]; exportName: string }) {
  const [f, setF] = useQueryStates(postFilterParsers, { history: "replace", scroll: false });
  const types = React.useMemo(() => [...new Set(rows.map((r) => r.type))].sort(), [rows]);
  const filtered = React.useMemo(() => filterPosts(rows, f), [rows, f]);
  const active = f.type != null || f.day != null || f.hour != null;
  const sort: PostSort = f.sort && (POST_SORTS as readonly string[]).includes(f.sort) ? (f.sort as PostSort) : "publishedAt";

  const toolbar = (
    <div className="flex flex-wrap items-center gap-2">
      <FilterSelect
        label={t.social.type}
        value={f.type ?? ALL}
        onChange={(v) => void setF({ type: v === ALL ? null : v })}
        options={types.map((ty) => ({ value: ty, label: ty }))}
      />
      <FilterSelect
        label={s.day}
        value={f.day == null ? ALL : String(f.day)}
        onChange={(v) => void setF({ day: v === ALL ? null : Number(v) })}
        options={s.days.map((d, i) => ({ value: String(i), label: d }))}
      />
      <FilterSelect
        label={s.hour}
        value={f.hour == null ? ALL : String(f.hour)}
        onChange={(v) => void setF({ hour: v === ALL ? null : Number(v) })}
        options={HOURS.map((h) => ({ value: String(h), label: `${String(h).padStart(2, "0")}:00` }))}
      />
      {active ? (
        <Button variant="ghost" size="sm" onClick={() => void setF({ type: null, day: null, hour: null })}>
          <X className="size-3.5" /> {s.resetFilters}
        </Button>
      ) : null}
      <span className="text-xs text-muted-foreground tabular" aria-live="polite">
        {formatNumber(filtered.length)} {t.common.of} {formatNumber(rows.length)} {s.post}
      </span>
    </div>
  );

  return (
    <PostsTable
      key={sort}
      rows={filtered}
      exportName={exportName}
      toolbar={toolbar}
      initialSort={sort}
    />
  );
}

function FilterSelect({
  label,
  value,
  onChange,
  options,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  options: { value: string; label: string }[];
}) {
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger size="sm" className="w-auto min-w-28 bg-background" aria-label={label}>
        <span className="text-muted-foreground">{label}:</span>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={ALL}>{t.common.all}</SelectItem>
        {options.map((o) => (
          <SelectItem key={o.value} value={o.value}>
            {o.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
