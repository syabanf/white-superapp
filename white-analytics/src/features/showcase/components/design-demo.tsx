"use client";

import { ArrowRight } from "lucide-react";
import { StatusBadge, type StatusKind } from "@/components/dashboard/status-badge";
import { Card } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { DESIGN_ORDERS, REVISIONS, VIDEO_STEPS, designStatus, type DesignStatus } from "../data";
import { DemoTable, Panel } from "./shared";

const STATUS: Record<DesignStatus, { label: string; kind: StatusKind }> = {
  done: { label: "Selesai", kind: "good" },
  onTrack: { label: "Sesuai jadwal", kind: "info" },
  atRisk: { label: "Berisiko telat", kind: "warning" },
  late: { label: "Terlambat", kind: "critical" },
};

const deadline = (days: number) => (days < 0 ? `Lewat ${-days} hari` : `${days} hari lagi`);

export function DesignDemo() {
  const totalCredits = VIDEO_STEPS.reduce((sum, s) => sum + s.credits, 0);
  const flagged = DESIGN_ORDERS.filter((o) => ["atRisk", "late"].includes(designStatus(o)));

  return (
    <Tabs defaultValue="recap" className="flex flex-col gap-(--gap)">
      <TabsList aria-label="Bagian" className="max-w-full overflow-x-auto">
        <TabsTrigger value="recap">Rekap file</TabsTrigger>
        <TabsTrigger value="revisions">Revisi klien</TabsTrigger>
        <TabsTrigger value="video">Generate dan edit video</TabsTrigger>
      </TabsList>

      <TabsContent value="recap" className="flex flex-col gap-(--gap)">
        <Panel title="Rekap desain per pesanan" hint={`AI mencocokkan file dengan target. ${flagged.length} pesanan perlu perhatian.`}>
          <DemoTable head={[{ label: "Pesanan" }, { label: "Desainer" }, { label: "Diunggah", right: true }, { label: "Disetujui", right: true }, { label: "Deadline" }, { label: "Status" }]}>
            {DESIGN_ORDERS.map((o) => {
              const s = STATUS[designStatus(o)];
              return (
                <tr key={`${o.client}-${o.item}`}>
                  <td className="px-4 py-3 whitespace-nowrap">
                    <div className="font-medium">{o.item}</div>
                    <div className="text-xs text-muted-foreground">{o.client}</div>
                  </td>
                  <td className="px-4 py-3">{o.owner}</td>
                  <td className="px-4 py-3 text-right tabular">
                    {o.uploaded} / {o.target}
                  </td>
                  <td className="px-4 py-3 text-right tabular">
                    {o.approved} / {o.target}
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">{deadline(o.daysLeft)}</td>
                  <td className="px-4 py-3">
                    <StatusBadge kind={s.kind}>{s.label}</StatusBadge>
                  </td>
                </tr>
              );
            })}
          </DemoTable>
        </Panel>
      </TabsContent>

      <TabsContent value="revisions" className="flex flex-col gap-(--gap)">
        <Panel title="Antrean revisi" hint="Revisi dari grup WhatsApp dan dashboard masuk ke satu daftar">
          <DemoTable head={[{ label: "File" }, { label: "Catatan klien" }, { label: "Sumber" }, { label: "Revisi" }, { label: "PIC" }, { label: "Status" }]}>
            {REVISIONS.map((r) => (
              <tr key={r.file}>
                <td className="px-4 py-3 whitespace-nowrap">
                  <div className="font-medium">{r.file}</div>
                  <div className="text-xs text-muted-foreground">{r.client}</div>
                </td>
                <td className="min-w-56 px-4 py-3">{r.note}</td>
                <td className="px-4 py-3 whitespace-nowrap text-muted-foreground">{r.source}</td>
                <td className="px-4 py-3 whitespace-nowrap tabular">{r.round}</td>
                <td className="px-4 py-3">{r.owner}</td>
                <td className="px-4 py-3">
                  <StatusBadge kind={r.state === "Kuota habis" ? "critical" : r.state === "Baru" ? "info" : "warning"}>{r.state}</StatusBadge>
                </td>
              </tr>
            ))}
          </DemoTable>
        </Panel>
      </TabsContent>

      <TabsContent value="video" className="flex flex-col gap-(--gap)">
        <div className="grid gap-(--gap) md:grid-cols-2 xl:grid-cols-4">
          {VIDEO_STEPS.map((s, i) => (
            <Card key={s.title} className="gap-2 p-(--card-pad)">
              <p className="label-mono flex items-center gap-1.5 text-brand">
                {String(i + 1).padStart(2, "0")} {i < VIDEO_STEPS.length - 1 ? <ArrowRight className="size-3" /> : null}
              </p>
              <h2 className="font-semibold">{s.title}</h2>
              <p className="text-sm text-muted-foreground">{s.note}</p>
              <p className="mt-auto pt-2 text-xs text-muted-foreground">{s.tool}</p>
              <p className="text-xs font-medium tabular">{s.credits ? `${s.credits} kredit` : "Tanpa kredit AI"}</p>
            </Card>
          ))}
        </div>
        <Panel title="Soal biaya token">
          <div className="flex flex-col gap-2 p-(--card-pad) text-sm text-muted-foreground">
            <p>
              Model teks seperti DeepSeek hanya menyusun naskah dan daftar potongan. Rendering video butuh layanan terpisah, jadi biayanya ikut terpisah. Contoh di atas memakai {totalCredits} kredit per video, angka ilustrasi yang belum memakai harga sungguhan.
            </p>
            <p>Fitur edit video belum ada di produk, jadi belum ada contoh hasil video asli. Editor tetap menyelesaikan video di Adobe Premiere.</p>
          </div>
        </Panel>
      </TabsContent>
    </Tabs>
  );
}
