import type { ReactNode } from "react";
import { FlaskConical } from "lucide-react";
import { Card } from "@/components/ui/card";

/** Every demo carries this, so nobody mistakes a simulation for live data. */
export function SimulationNotice({ live, ai }: { live: string; ai?: boolean }) {
  return (
    <div className="flex items-start gap-3 rounded-2xl border border-brand/20 bg-brand/5 px-4 py-3 text-sm">
      <FlaskConical className="mt-0.5 size-4 shrink-0 text-brand" />
      <p className="min-w-0">
        <span className="font-semibold">Simulasi dengan data contoh.</span>{" "}
        {ai
          ? "Tidak ada pesan WhatsApp yang dikirim. Balasan dibuat AI dari teks yang Anda ketik, dan teks itu terkirim ke penyedia AI, jadi jangan ketik data klien asli."
          : "Tidak ada pesan yang dikirim dan tidak ada koneksi ke layanan luar."}{" "}
        <span className="text-muted-foreground">{live}</span>
      </p>
    </div>
  );
}

/** Titled card used by every demo panel. */
export function Panel({ title, hint, action, children }: { title: string; hint?: string; action?: ReactNode; children: ReactNode }) {
  return (
    <Card className="gap-0 overflow-hidden py-0">
      <header className="flex flex-wrap items-center justify-between gap-2 border-b px-(--card-pad) py-3.5">
        <div className="min-w-0">
          <h2 className="text-sm font-semibold">{title}</h2>
          {hint ? <p className="mt-0.5 text-xs text-muted-foreground">{hint}</p> : null}
        </div>
        {action}
      </header>
      {children}
    </Card>
  );
}

/** Scrollable table shell; rows bring their own cells. */
export function DemoTable({ head, children }: { head: { label: string; right?: boolean }[]; children: ReactNode }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead className="border-b bg-muted/40 text-xs text-muted-foreground">
          <tr>
            {head.map((h) => (
              <th key={h.label} className={`px-4 py-2.5 font-medium whitespace-nowrap ${h.right ? "text-right" : "text-left"}`}>
                {h.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y">{children}</tbody>
      </table>
    </div>
  );
}
