"use client";

import * as React from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { Fld, SimpleSelect, useFieldErrors } from "@/features/erp/hr/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { formatDateTime } from "@/lib/format";
import { saveScreeningAction } from "../actions";
import { toRupiahInt } from "../lib";
import { RECOMMENDATION, rs } from "../strings";
import type { Screening } from "../types";

const s = rs.screening;
const TRI = [
  { value: "yes", label: rs.yes },
  { value: "no", label: rs.no },
];
const toTri = (b: boolean | undefined) => (b === undefined ? "" : b ? "yes" : "no");
const fromTri = (v: string) => (v === "" ? undefined : v === "yes");

/** Phone screening form (NüHabit screening call). */
export function ScreeningForm({ candidateId, screening, canWrite }: { candidateId: string; screening: Screening | null; canWrite: boolean }) {
  const [v, setV] = React.useState({
    contacted: screening?.contacted ?? false,
    interested: toTri(screening?.interested),
    availabilityNote: screening?.availabilityNote ?? "",
    confirmedSalary: screening?.confirmedSalary ?? "",
    willingShift: toTri(screening?.willingShift),
    willingPlacement: toTri(screening?.willingPlacement),
    notes: screening?.notes ?? "",
    recommendation: screening?.recommendation ?? "",
  });
  const set = <K extends keyof typeof v>(k: K, val: (typeof v)[K]) => setV((p) => ({ ...p, [k]: val }));
  const { errors, onFail } = useFieldErrors();
  const { run, pending } = useErpAction();
  const disabled = !canWrite || pending;

  return (
    <Card>
      <CardHeader>
        <CardTitle>{s.title}</CardTitle>
        {screening?.updatedByName ? (
          <p className="text-xs text-muted-foreground">{s.updatedBy(screening.updatedByName, formatDateTime(screening.updatedAt))}</p>
        ) : null}
      </CardHeader>
      <CardContent>
        <form
          className="grid gap-4 sm:grid-cols-2"
          onSubmit={(e) => {
            e.preventDefault();
            void run(
              () =>
                saveScreeningAction(candidateId, {
                  ...v,
                  interested: fromTri(v.interested),
                  willingShift: fromTri(v.willingShift),
                  willingPlacement: fromTri(v.willingPlacement),
                  recommendation: v.recommendation as "" | "lolos" | "hold" | "tidak_lolos",
                }),
              { success: s.saved, onFail },
            );
          }}
        >
          <div className="flex items-center gap-2 sm:col-span-2">
            <Switch id="scr-contacted" checked={v.contacted} onCheckedChange={(x) => set("contacted", x)} disabled={disabled} />
            <Label htmlFor="scr-contacted">{s.contacted}</Label>
          </div>
          <Fld label={s.interested} htmlFor="scr-int">
            <SimpleSelect id="scr-int" value={v.interested} onChange={(x) => set("interested", x)} options={TRI} emptyLabel={rs.unknown} disabled={disabled} />
          </Fld>
          <Fld label={s.confirmedSalary} htmlFor="scr-sal" error={errors?.confirmedSalary}>
            <MoneyInput id="scr-sal" value={v.confirmedSalary} onChange={(x) => set("confirmedSalary", toRupiahInt(x) ?? "")} disabled={disabled} />
          </Fld>
          <Fld label={s.willingShift} htmlFor="scr-shift">
            <SimpleSelect id="scr-shift" value={v.willingShift} onChange={(x) => set("willingShift", x)} options={TRI} emptyLabel={rs.unknown} disabled={disabled} />
          </Fld>
          <Fld label={s.willingPlacement} htmlFor="scr-place">
            <SimpleSelect id="scr-place" value={v.willingPlacement} onChange={(x) => set("willingPlacement", x)} options={TRI} emptyLabel={rs.unknown} disabled={disabled} />
          </Fld>
          <Fld label={s.availabilityNote} htmlFor="scr-av" error={errors?.availabilityNote} className="sm:col-span-2">
            <Input id="scr-av" value={v.availabilityNote} onChange={(e) => set("availabilityNote", e.target.value)} disabled={disabled} maxLength={200} />
          </Fld>
          <Fld label={s.notes} htmlFor="scr-notes" error={errors?.notes} className="sm:col-span-2">
            <Textarea id="scr-notes" rows={3} value={v.notes} onChange={(e) => set("notes", e.target.value)} disabled={disabled} />
          </Fld>
          <Fld label={s.recommendation} htmlFor="scr-rec">
            <SimpleSelect
              id="scr-rec"
              value={v.recommendation}
              onChange={(x) => set("recommendation", x)}
              options={Object.entries(RECOMMENDATION).map(([value, label]) => ({ value, label }))}
              emptyLabel={s.noRecommendation}
              disabled={disabled}
            />
          </Fld>
          {canWrite ? (
            <div className="flex items-end justify-end">
              <Button type="submit" disabled={pending}>
                {pending ? <Loader2 className="size-4 animate-spin" /> : null}
                {rs.save}
              </Button>
            </div>
          ) : null}
        </form>
      </CardContent>
    </Card>
  );
}
