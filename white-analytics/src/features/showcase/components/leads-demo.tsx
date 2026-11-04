"use client";

import * as React from "react";
import { KpiTile } from "@/components/dashboard/kpi-tile";
import { StatusBadge, type StatusKind } from "@/components/dashboard/status-badge";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { formatCurrency, formatNumber } from "@/lib/format";
import { WA_FUNNEL, WA_LEADS, leadQuality, type LeadQuality } from "../data";
import { DemoTable, Panel } from "./shared";

const QUALITY: Record<LeadQuality, { label: string; kind: StatusKind }> = {
  qualified: { label: "Berkualitas", kind: "good" },
  review: { label: "Perlu dicek", kind: "warning" },
  notTarget: { label: "Bukan target", kind: "neutral" },
};

/** Chats out of WA_FUNNEL.chats counted as qualified in this example. */
const QUALIFIED_CHATS = 74;

export function LeadsDemo() {
  const [filter, setFilter] = React.useState<LeadQuality | "all">("all");
  const rows = WA_LEADS.filter((l) => filter === "all" || leadQuality(l.score) === filter);

  return (
    <>
      <div className="grid gap-(--gap) sm:grid-cols-2 xl:grid-cols-4">
        <KpiTile label="Klik iklan" value={formatNumber(WA_FUNNEL.clicks)} caption="Iklan traffic ke WhatsApp" />
        <KpiTile label="Chat masuk" value={formatNumber(WA_FUNNEL.chats)} caption="Nomor tercatat otomatis" />
        <KpiTile label="Lead berkualitas" value={formatNumber(QUALIFIED_CHATS)} caption="Dinilai dari isi chat" />
        <KpiTile label="Biaya per lead berkualitas" value={formatCurrency(WA_FUNNEL.spend / QUALIFIED_CHATS)} caption={`Belanja ${formatCurrency(WA_FUNNEL.spend)}`} />
      </div>

      <Panel
        title="Chat terbaru dari iklan"
        hint="Contoh 7 chat. Nomor disamarkan di demo."
        action={
          <Tabs value={filter} onValueChange={(v) => setFilter(v as LeadQuality | "all")}>
            <TabsList aria-label="Saring kualitas" className="max-w-full overflow-x-auto">
              <TabsTrigger value="all">Semua</TabsTrigger>
              {(Object.keys(QUALITY) as LeadQuality[]).map((q) => (
                <TabsTrigger key={q} value={q}>
                  {QUALITY[q].label}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        }
      >
        <DemoTable head={[{ label: "Customer" }, { label: "Dari iklan" }, { label: "Pesan pertama" }, { label: "Skor", right: true }, { label: "Kualitas" }]}>
          {rows.map((l) => {
            const q = QUALITY[leadQuality(l.score)];
            return (
              <tr key={l.phone}>
                <td className="px-4 py-3 align-top whitespace-nowrap">
                  <div className="font-medium">{l.name}</div>
                  <div className="text-xs text-muted-foreground tabular">{l.phone}</div>
                </td>
                <td className="px-4 py-3 align-top text-muted-foreground">{l.campaign}</td>
                <td className="min-w-64 px-4 py-3 align-top">{l.message}</td>
                <td className="px-4 py-3 text-right align-top font-medium tabular">{l.score}</td>
                <td className="px-4 py-3 align-top">
                  <StatusBadge kind={q.kind}>{q.label}</StatusBadge>
                </td>
              </tr>
            );
          })}
        </DemoTable>
      </Panel>

      <Panel title="Bagaimana dengan iklan awareness?">
        <p className="p-(--card-pad) text-sm text-muted-foreground">
          Meta tidak memberikan nomor orang yang melihat iklan. Nomor hanya bisa didapat kalau customer mengirim chat, jadi iklan awareness diukur dari jangkauan dan frekuensi. Iklan traffic bisa diarahkan ke WhatsApp seperti contoh di atas.
        </p>
      </Panel>
    </>
  );
}
