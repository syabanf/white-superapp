"use client";

import * as React from "react";
import { Sparkles } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { formatCurrency, formatDecimal, formatNumber } from "@/lib/format";
import { MARKETPLACE_BRANDS, MARKETPLACE_INSIGHTS, brandSales, type Marketplace } from "../data";
import { DemoTable, Panel } from "./shared";

const TABS: { value: Marketplace | "all"; label: string }[] = [
  { value: "all", label: "Semua" },
  { value: "shopee", label: "Shopee" },
  { value: "tokopedia", label: "Tokopedia" },
  { value: "tiktok", label: "TikTok Shop" },
];

export function MarketplaceDemo() {
  const [market, setMarket] = React.useState<Marketplace | "all">("all");
  const rows = MARKETPLACE_BRANDS.map((b) => ({ ...b, ...brandSales(b, market) })).sort((a, b) => b.revenue - a.revenue);
  const top = rows[0].revenue;

  return (
    <>
      <Panel
        title="Perbandingan 30 hari terakhir"
        hint="Kategori kopi bubuk, merek pembanding fiktif"
        action={
          <Tabs value={market} onValueChange={(v) => setMarket(v as Marketplace | "all")}>
            <TabsList aria-label="Marketplace" className="max-w-full overflow-x-auto">
              {TABS.map((t) => (
                <TabsTrigger key={t.value} value={t.value}>
                  {t.label}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        }
      >
        <DemoTable head={[{ label: "Merek" }, { label: "Perkiraan omzet" }, { label: "Terjual", right: true }, { label: "Harga", right: true }, { label: "Rating", right: true }, { label: "Video", right: true }]}>
          {rows.map((b) => (
            <tr key={b.name} className={b.own ? "bg-brand/5" : undefined}>
              <td className="px-4 py-3 whitespace-nowrap">
                <span className="font-medium">{b.name}</span> {b.own ? <Badge variant="secondary">Klien</Badge> : null}
              </td>
              <td className="min-w-56 px-4 py-3">
                <div className="tabular">{formatCurrency(b.revenue)}</div>
                <div className="mt-1 h-1.5 rounded-full bg-muted">
                  <div className="h-full rounded-full bg-brand" style={{ width: `${(b.revenue / top) * 100}%` }} />
                </div>
              </td>
              <td className="px-4 py-3 text-right tabular">{formatNumber(b.units)}</td>
              <td className="px-4 py-3 text-right tabular">{formatCurrency(b.price)}</td>
              <td className="px-4 py-3 text-right tabular">{formatDecimal(b.rating)}</td>
              <td className="px-4 py-3 text-right tabular">{b.videos}</td>
            </tr>
          ))}
        </DemoTable>
      </Panel>

      <Panel title="Analisa AI" hint="Contoh analisa dari data di atas (semua marketplace)">
        <ul className="flex flex-col gap-3 p-(--card-pad)">
          {MARKETPLACE_INSIGHTS.map((text) => (
            <li key={text} className="flex gap-2.5 text-sm">
              <Sparkles className="mt-0.5 size-4 shrink-0 text-brand" />
              <span>{text}</span>
            </li>
          ))}
        </ul>
      </Panel>

      <Panel title="Sumber data">
        <p className="p-(--card-pad) text-sm text-muted-foreground">
          Angka di atas contoh. Data asli butuh penyedia pihak ketiga seperti Kalodata atau alternatif lain. Cakupan marketplace, harga, dan akses API tiap penyedia perlu dibandingkan dulu sebelum dipilih.
        </p>
      </Panel>
    </>
  );
}
