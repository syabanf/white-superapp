"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { toast } from "@/lib/toast";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { setAuditCadence } from "@/features/seo/actions";
import { s } from "@/features/seo-suite/strings";

type Cadence = "NONE" | "WEEKLY" | "DAILY";
const LABELS: Record<Cadence, string> = { NONE: s.cadenceNone, WEEKLY: s.cadenceWeekly, DAILY: s.cadenceDaily };

/** "Jadwal audit" — how often the daily cron re-runs the audit for this property. */
export function AuditCadenceSelect({ clientId, propertyId, value }: { clientId: string; propertyId: string; value: Cadence }) {
  const router = useRouter();
  const [pending, startTransition] = React.useTransition();
  const [current, setCurrent] = React.useState<Cadence>(value);

  const change = (next: Cadence) => {
    const prev = current;
    setCurrent(next);
    startTransition(async () => {
      const res = await setAuditCadence({ clientId, propertyId, cadence: next });
      if (res.ok) {
        toast.success(s.cadenceSaved, {
          description: LABELS[next],
          action: {
            label: "Urungkan",
            onClick: () => {
              setCurrent(prev);
              startTransition(async () => {
                const undo = await setAuditCadence({ clientId, propertyId, cadence: prev });
                if (undo.ok) {
                  toast.success("Jadwal audit dikembalikan", { description: LABELS[prev] });
                  router.refresh();
                } else toast.error(undo.error);
              });
            },
          },
        });
        router.refresh();
      } else {
        setCurrent(prev);
        toast.error(res.error);
      }
    });
  };

  return (
    <div className="flex items-center gap-2">
      <Label htmlFor="audit-cadence" className="label-mono whitespace-nowrap">
        {s.auditSchedule}
      </Label>
      <Select value={current} onValueChange={(v) => change(v as Cadence)} disabled={pending}>
        <SelectTrigger id="audit-cadence" size="sm" className="min-w-28">
          {pending ? <Spinner className="size-3.5" /> : null}
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {(Object.keys(LABELS) as Cadence[]).map((c) => (
            <SelectItem key={c} value={c}>
              {LABELS[c]}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
