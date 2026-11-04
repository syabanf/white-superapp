"use client";

import * as React from "react";
import { Check, Sparkles, X } from "lucide-react";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { AD_RECOMMENDATIONS } from "../data";

type Decision = "approved" | "rejected";

/** Approval queue: the AI proposes, a person decides, nothing changes on Meta in this demo. */
export function AdsDemo() {
  const [decisions, setDecisions] = React.useState<Record<string, Decision>>({});
  const decide = (id: string, d: Decision) => setDecisions((all) => ({ ...all, [id]: d }));
  const approved = AD_RECOMMENDATIONS.filter((r) => decisions[r.id] === "approved");

  return (
    <>
      <div className="grid gap-(--gap) lg:grid-cols-2">
        {AD_RECOMMENDATIONS.map((r) => {
          const d = decisions[r.id];
          return (
            <Card key={r.id} className="gap-3 p-(--card-pad)">
              <div className="flex items-start justify-between gap-3">
                <p className="text-xs text-muted-foreground">{r.campaign}</p>
                <span className="shrink-0 text-xs text-muted-foreground tabular">Keyakinan {r.confidence}%</span>
              </div>
              <h2 className="flex gap-2 font-semibold">
                <Sparkles className="mt-0.5 size-4 shrink-0 text-brand" /> {r.action}
              </h2>
              <p className="text-sm text-muted-foreground">{r.why}</p>
              <p className="text-sm">
                <span className="font-medium">Perkiraan dampak:</span> {r.impact}
              </p>
              {d ? (
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <StatusBadge kind={d === "approved" ? "good" : "neutral"}>{d === "approved" ? "Disetujui" : "Ditolak"}</StatusBadge>
                  <Button size="sm" variant="ghost" onClick={() => setDecisions((all) => Object.fromEntries(Object.entries(all).filter(([key]) => key !== r.id)))}>
                    Batalkan
                  </Button>
                </div>
              ) : (
                <div className="flex flex-wrap gap-2">
                  <Button size="sm" onClick={() => decide(r.id, "approved")}>
                    <Check className="size-4" /> Setujui
                  </Button>
                  <Button size="sm" variant="outline" onClick={() => decide(r.id, "rejected")}>
                    <X className="size-4" /> Tolak
                  </Button>
                </div>
              )}
            </Card>
          );
        })}
      </div>

      <p aria-live="polite" className="text-sm text-muted-foreground">
        {approved.length
          ? `${approved.length} tindakan disetujui. Saat terhubung, sistem menjalankannya di Meta dan mencatat siapa yang menyetujui. Di demo ini tidak ada yang berubah.`
          : "Budget klien tidak berubah tanpa persetujuan tim."}
      </p>
    </>
  );
}
