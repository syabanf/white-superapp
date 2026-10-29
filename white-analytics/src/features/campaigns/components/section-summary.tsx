"use client";

import { CheckCircle2, CircleAlert } from "lucide-react";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { InfoHint } from "@/components/dashboard/info-hint";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { formatCompact, formatCurrency, formatDate } from "@/lib/format";
import { estimateDailyReach, filledCreatives, scheduleDays, type DraftIssue } from "@/features/campaigns/lib";
import type { WriterGate } from "@/lib/providers/meta-ads-writer/types";
import { c, objectiveLabel, placementLabel } from "@/features/campaigns/strings";
import type { BuilderState, SetField } from "./builder-state";
import { Block } from "./block";
import { SimulatedBadge } from "./status";

function Row({ label, children }: { label: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5 py-2.5 sm:flex-row sm:items-baseline sm:justify-between sm:gap-4">
      <dt className="flex items-center gap-1.5 text-sm text-muted-foreground">{label}</dt>
      <dd className="min-w-0 text-sm font-medium sm:text-right">{children}</dd>
    </div>
  );
}

export function SectionSummary({
  state,
  set,
  currency,
  gate,
  issues,
  disabled,
  submitted,
}: {
  state: BuilderState;
  set: SetField;
  currency: string;
  gate: WriterGate;
  issues: DraftIssue[];
  disabled: boolean;
  submitted: { externalId: string | null; simulated: boolean; submittedAt: string | null } | null;
}) {
  const days = scheduleDays(state.startDate, state.endDate || null);
  const reach = estimateDailyReach(state.dailyBudget, currency);
  const a = state.audience;
  const messages = Array.from(new Set(issues.map((i) => i.message)));
  return (
    <Block id="step-summary" index="05" title={c.sectionSummary}>
      {submitted ? (
        <div className="mb-4 space-y-2 rounded-2xl bg-muted/60 p-4">
          <div className="flex flex-wrap items-center gap-2">
            <StatusBadge kind="good">{c.status.SUBMITTED}</StatusBadge>
            {submitted.simulated ? <SimulatedBadge /> : <StatusBadge kind="info">{c.realBadge}</StatusBadge>}
          </div>
          <p className="text-sm">{submitted.simulated ? c.submittedNote : c.submittedRealNote}</p>
          {submitted.externalId ? (
            <p className="text-xs text-muted-foreground">
              {c.externalIdLabel}: <code className="tabular">{submitted.externalId}</code>
              {submitted.submittedAt ? ` · ${formatDate(submitted.submittedAt)}` : null}
            </p>
          ) : null}
        </div>
      ) : null}

      <dl className="divide-y divide-border/60">
        <Row label={c.objectiveLabel}>{objectiveLabel(state.objective)}</Row>
        <Row label={c.budgetLabel}>
          {formatCurrency(state.dailyBudget, currency)}
          {c.perDay}
        </Row>
        <Row label={c.totalBudgetLabel}>{days ? `${formatCurrency(state.dailyBudget * days, currency)} · ${c.days(days)}` : c.openEnded}</Row>
        <Row label={c.sectionAudience}>
          {[a.locations.join(", ") || "–", `${a.ageMin}–${a.ageMax}`, c.genders[a.genders]].join(" · ")}
          {a.interests.length ? <span className="block text-xs font-normal text-muted-foreground">{a.interests.join(", ")}</span> : null}
        </Row>
        <Row label={c.sectionPlacements}>{state.placements.length ? state.placements.map(placementLabel).join(", ") : "Advantage+"}</Row>
        <Row label={c.sectionCreatives}>{c.creativesCount(filledCreatives(state.creatives).length)}</Row>
        <Row
          label={
            <>
              {c.reachLabel} <InfoHint>{c.reachHint}</InfoHint>
            </>
          }
        >
          {reach ? c.reachValue(formatCompact(reach.low), formatCompact(reach.high)) : c.reachUnknown}
        </Row>
        {!submitted ? (
          <Row label={c.modeLabel}>
            {gate.mode === "real" ? (
              <StatusBadge kind="info">{c.modeReal}</StatusBadge>
            ) : (
              <span className="flex flex-col items-start gap-1 sm:items-end">
                <SimulatedBadge />
                <span className="text-xs font-normal text-muted-foreground">{gate.reasons.map((r) => c.gateReasons[r] ?? r).join(" · ")}</span>
              </span>
            )}
          </Row>
        ) : null}
      </dl>

      {!submitted ? (
        <div className="mt-4 rounded-2xl bg-muted/60 p-4" aria-live="polite">
          {messages.length ? (
            <>
              <h3 className="mb-1.5 flex items-center gap-1.5 text-sm font-semibold">
                <CircleAlert className="size-4 text-[color:var(--status-warning)]" /> {c.issuesTitle}
              </h3>
              <ul className="list-disc space-y-0.5 pl-5 text-sm text-muted-foreground">
                {messages.map((m) => (
                  <li key={m}>{m}</li>
                ))}
              </ul>
            </>
          ) : (
            <p className="flex items-center gap-1.5 text-sm font-medium">
              <CheckCircle2 className="size-4 text-[color:var(--status-good)]" /> {c.noIssues}
            </p>
          )}
        </div>
      ) : null}

      <div className="mt-4 space-y-1.5">
        <Label htmlFor="cmp-notes">{c.aiNotesLabel}</Label>
        <Textarea id="cmp-notes" value={state.aiNotes} onChange={(e) => set("aiNotes", e.target.value)} placeholder={c.aiNotesEmpty} rows={4} maxLength={4000} disabled={disabled} />
      </div>
    </Block>
  );
}
