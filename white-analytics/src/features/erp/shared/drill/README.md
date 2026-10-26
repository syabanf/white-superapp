# Drill-down and growth coloring

Shared by ERP and marketing dashboards. Owned by the primitives agent: do not edit these files or `src/components/dashboard/**`; ask for changes instead.

```ts
import { DrillSheet, DrillBreadcrumb, useDrill, useDrillParams, withParams, timeDrill, bucketLabel } from "@/features/erp/shared/drill";
```

`drill/href.ts` and `drill/time.ts` are pure, so server pages can import them directly (`@/features/erp/shared/drill/href`). `DrillSheet`, `useDrill` and `useDrillParams` are client-only.

## 1. Drill sheet (default for chart marks)

URL state: `?drill=<kind>:<key>`. `open()` pushes a history entry, so browser Back closes the sheet. The key may contain colons (`cell:2:4`).

```tsx
"use client";
const { open } = useDrill();

<TimeSeriesChart data={rows} series={series} onSelect={(p) => open("revenue-month", p.date)} />

<DrillSheet
  kind="revenue-month"                    // or ["revenue-month", "cost-month"]
  title={({ key }) => `Jurnal ${bucketLabel(key, "month")}`}
  chips={({ key }) => [bucketLabel(key, "month"), deptLabel].filter(Boolean)}
  actions={({ key }) => {
    const next = timeDrill(key, "month");   // optional time drill from inside the sheet
    return next ? [{ label: "Lihat per hari", href: hrefWith({ from: next.from, to: next.to }) }] : [];
  }}
  load={async ({ key }, signal) => {
    const { from, to } = bucketRange(key, "month");
    const res = await erpClientFetch<JournalList>(`/v1/finance/journals?from=${from}&to=${to}&pageSize=20`, { signal });
    return {
      summary: [{ label: "Jumlah", value: formatNumber(Number(res.total)) }, { label: "Total", value: formatIDR(sum) }],
      total: Number(res.total),            // shows "Menampilkan 20 dari 134"
      items: res.items.map((j) => ({
        id: j.id,
        title: j.description,
        subtitle: formatDay(j.date),
        value: formatIDR(j.amount),
        meta: <DeltaText current={a} previous={b} />,   // optional
        badge: <StatusBadge … />,                       // optional
        href: `/erp/finance/journals/${j.id}`,
      })),
    };
  }}
  fullListHref={({ key }) => { const r = bucketRange(key, "month"); return `/erp/finance/journals?from=${r.from}&to=${r.to}`; }}
/>
```

- Item `href`: internal paths use Next `Link`; absolute `http(s)://` URLs (permalinks, backlink sources) open in a new tab with `rel="noopener noreferrer"`. Set `external: true` to force a new tab for a same-origin URL.
- `load` may return synchronously when the rows are already on the client (`load={({ key }) => ({ items: rows.filter(...) })}`). A thrown `Error` shows its message with a retry button. Loading, empty and error states are built in.
- Mount one `DrillSheet` per kind (or one with several kinds) anywhere inside the dashboard's client tree. It renders right-side on desktop (max 28rem) and as a bottom sheet (max 85dvh, internal scroll) on phones.
- `fullListLabel` defaults to "Buka daftar lengkap"; `emptyLabel` to "Tidak ada data untuk bagian ini".
- Custom state handling: `DrillSheetView` is the same panel, controlled (`open`, `onOpenChange`, `state: {status: "loading" | "error" | "ready"}`).
- `useDrill()` returns `{ drill, open(kind, key), close(), hrefFor(kind, key), isOpen(kind) }`. Use `hrefFor` when a plain `<Link>` should open a sheet (e.g. a table cell).

## 2. In-dashboard drill (time levels, dimension filters)

```tsx
const { apply, hrefWith, get } = useDrillParams();

// time mark: year -> month -> day
onSelect={(p) => { const next = timeDrill(p.date, bucket); if (next) apply({ from: next.from, to: next.to, preset: null, bucket: next.bucket }); }}
// dimension mark: apply as a dashboard filter
onSelect={(item) => apply({ dept: item.key })}

<DrillBreadcrumb
  items={[
    { label: "Semua", href: hrefWith({ from: null, to: null, dept: null }) },
    { label: "Maret 2026", href: hrefWith({ dept: null }) },
    { label: "Creative" },                  // current level, no href
  ]}
/>
```

`apply` pushes a new history entry (Back goes up a level) and re-renders the server page; it drops `page` and `drill` by default. Server pages build the same links with `withParams(pathname, searchParams, patch)`.

Time helpers (`drill/time.ts`): `bucketRange(start, bucket)` gives the inclusive `{from, to}`; `timeDrill(start, bucket)` gives the range plus the finer bucket (year/quarter -> month -> day, week -> day, null at day); `bucketLabel` gives "2026", "K1 2026", "Maret 2026", "Minggu 2 Mar 2026", "2 Mar 2026"; `bucketForRange(from, to)` picks day/month/year.

Sheet vs in-dashboard: categorical marks open the sheet; trends time-drill. When both help, open the sheet and add the other route in `actions` (e.g. "Lihat per hari", "Saring dasbor ke Creative").

## 3. Click support on chart primitives (`src/components/dashboard/charts`)

| Primitive | Click API |
| --- | --- |
| `TimeSeriesChart` | `onSelect(point, index)` on any point/bar; `selectLabel`; legend filter `onSeriesSelect(series)` + `activeSeries` |
| `StackedBarChart` | `onSelect(row, seriesKey \| null, index)` (series key when a segment was hit); row `href` still works; `onSeriesSelect` + `activeSeries` |
| `HBarChart` | item `href` (Next Link; `/path#anchor` scrolls to the anchor after navigation) or `onSelect(item, index)`; item `key` for ids; item `delta` + `lowerIsBetter` show colored change |
| `Heatmap` | `onSelect(cell)` with `{row, col, value, count}`; cells become buttons |
| `BoxPlot` | row `href` or `onSelect(row)`; `countLabel` for the tooltip noun |
| `BulletChart` | row `href` or `onSelect(row)` |
| `FunnelChart` | step `href` or `onSelect(step, index)` |
| `WaterfallChart` | `onSelect(step, index)` (`onStepClick` still works); `lowerIsBetter` for cost bridges |
| `ScoreRing` | `href` |
| `SeriesLegend` | `onSelect(series)` + `activeKey` (toggle buttons) |

All clickable marks get a pointer cursor and hover state. Recharts-based charts (`TimeSeriesChart`, `StackedBarChart`, `WaterfallChart`) also render a keyboard route (`MarkButtons`, visible on focus). For your own recharts charts use `activeIndexOf(state)` in the chart `onClick` and `MarkButtons` from `chart-primitives`.

## 4. Growth and decline coloring

Color means good vs bad, never up vs down, and always comes with an arrow or sign. Pass `lowerIsBetter` for cost, late rate, overdue, churn, DSO/DPO, response time, CPC/CPR, SEO position, days-to-hire and the like.

- `KpiTile`: `delta` or raw `currentValue` + `previousValue` (delta computed). The badge, the "Periode sebelumnya" line (`previous`) and the sparkline are tinted automatically. For a value that is itself a signed change (net follower growth "+1.148", net income), pass `signedValue={raw}` to color the figure (keep the sign in the formatted `value`).
- `StatStrip` items: same fields (`delta` or `currentValue`/`previousValue`, `previous`, `lowerIsBetter`).
- `DeltaBadge` (pill) and `DeltaText` (inline, for tables and captions): `delta` or `current` + `previous`, `mode="pct" | "abs"`, `format(delta)` for custom labels such as currency, `lowerIsBetter`.
- `DataTable` (`@/components/dashboard/data-table`): `initialSearch` prefills the search box (e.g. from `?q=`).
- Tables: `deltaColumn({ id, header, current, previous, lowerIsBetter, mode, format })` from `@/components/dashboard/data-table` gives a sortable colored "vs" column. In custom tables render `<DeltaText … />` in the cell.
- `Sparkline`: tinted by last vs first value (or `trend={delta}`), muted when flat; `stroke` opts out.
- Signed bars: `series: [{ key: "net", label: "Laba bersih", type: "bar", signed: true }]` in `TimeSeriesChart` (and non-stacked `StackedBarChart`); `lowerIsBetter` on the series flips it. For custom recharts bars use `signedFill(value, lowerIsBetter)` on `<Cell>`.
- Waterfall: increases positive, decreases negative, totals brand; `lowerIsBetter` flips. `waterfallFill(step, lowerIsBetter)` for custom bridges.
- Helpers in `@/lib/metrics`: `changeTone(deltaOrNumber, lowerIsBetter)` -> `"positive" | "negative" | "neutral"`, `trendColor()` -> CSS var, `trendTextClass()` -> `text-positive` / `text-negative` / `text-muted-foreground`, `seriesDelta(values)`.

Categorical series colors stay as they are; only change-encoding marks use positive/negative.
