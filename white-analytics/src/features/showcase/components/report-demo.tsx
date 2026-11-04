"use client";

import * as React from "react";
import { CheckCircle2, FileText, Loader2 } from "lucide-react";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { Button } from "@/components/ui/button";
import { formatCompact, formatCurrency, formatDecimal } from "@/lib/format";
import { BUZZER_ACCOUNTS, REPORT_PREVIEW, REPORT_STEPS, costPerThousand } from "../data";
import { DemoTable, Panel } from "./shared";

const STEP_MS = 900;

export function ReportDemo() {
  // null = not started; otherwise the number of finished steps.
  const [done, setDone] = React.useState<number | null>(null);
  const running = done !== null && done < REPORT_STEPS.length;

  React.useEffect(() => {
    if (!running) return;
    const timer = setTimeout(() => setDone((d) => (d ?? 0) + 1), STEP_MS);
    return () => clearTimeout(timer);
  }, [running, done]);

  return (
    <>
      <Panel
        title="Report konten otomatis"
        hint="Kopi Nusantara, Oktober"
        action={
          <Button size="sm" disabled={running} onClick={() => setDone(0)}>
            {running ? <Loader2 className="size-4 animate-spin" /> : <FileText className="size-4" />}
            {done === null ? "Buat report" : running ? "Menyusun..." : "Buat ulang"}
          </Button>
        }
      >
        <div aria-live="polite" className="flex flex-col gap-4 p-(--card-pad)">
          {done === null ? <p className="text-sm text-muted-foreground">Klik Buat report untuk melihat alurnya.</p> : null}
          {running ? (
            <ul className="flex flex-col gap-2 text-sm">
              {REPORT_STEPS.map((s, i) => (
                <li key={s} className={i > done ? "flex items-center gap-2 text-muted-foreground" : "flex items-center gap-2"}>
                  {i < done ? <CheckCircle2 className="size-4 text-positive" /> : i === done ? <Loader2 className="size-4 animate-spin text-brand" /> : <span className="size-4" />}
                  {s}
                </li>
              ))}
            </ul>
          ) : null}
          {done === REPORT_STEPS.length ? (
            <>
              <h3 className="font-semibold">{REPORT_PREVIEW.title}</h3>
              <dl className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                {REPORT_PREVIEW.stats.map((s) => (
                  <div key={s.label} className="rounded-2xl bg-muted/50 p-3">
                    <dt className="text-xs text-muted-foreground">{s.label}</dt>
                    <dd className="mt-1 text-lg font-semibold tabular">{s.value}</dd>
                  </div>
                ))}
              </dl>
              <div className="flex flex-col gap-1.5 text-sm">
                <p><span className="font-medium">Terbaik:</span> {REPORT_PREVIEW.best}</p>
                <p><span className="font-medium">Terlemah:</span> {REPORT_PREVIEW.worst}</p>
                <p><span className="font-medium">Saran AI:</span> {REPORT_PREVIEW.next}</p>
              </div>
            </>
          ) : null}
        </div>
      </Panel>

      <Panel title="Pantauan buzzer" hint="Kampanye promo kopi susu, akun fiktif">
        <DemoTable head={[{ label: "Akun" }, { label: "Followers", right: true }, { label: "Post", right: true }, { label: "Jangkauan", right: true }, { label: "Engagement", right: true }, { label: "Biaya per 1.000 orang", right: true }, { label: "Catatan AI" }]}>
          {BUZZER_ACCOUNTS.map((a) => (
            <tr key={a.handle}>
              <td className="px-4 py-3 whitespace-nowrap">
                <div className="font-medium">{a.handle}</div>
                <div className="text-xs text-muted-foreground">{a.tier}</div>
              </td>
              <td className="px-4 py-3 text-right tabular">{formatCompact(a.followers)}</td>
              <td className="px-4 py-3 text-right tabular">{a.posts}</td>
              <td className="px-4 py-3 text-right tabular">{formatCompact(a.reach)}</td>
              <td className="px-4 py-3 text-right tabular">{formatDecimal(a.engagement)}%</td>
              <td className="px-4 py-3 text-right tabular">{formatCurrency(costPerThousand(a))}</td>
              <td className="px-4 py-3">
                {a.suspicious ? <StatusBadge kind="warning">Interaksi janggal</StatusBadge> : <StatusBadge kind="good">Wajar</StatusBadge>}
              </td>
            </tr>
          ))}
        </DemoTable>
      </Panel>
    </>
  );
}
