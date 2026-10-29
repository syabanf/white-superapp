"use client";

import { Sparkles, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { InfoHint } from "@/components/dashboard/info-hint";
import { OBJECTIVES, minDailyBudget } from "@/features/campaigns/lib";
import { c } from "@/features/campaigns/strings";
import type { BuilderAdAccount } from "@/features/campaigns/queries";
import type { BuilderState, SetField } from "./builder-state";
import { Block, FieldIssues } from "./block";

type Props = {
  state: BuilderState;
  set: SetField;
  adAccounts: Pick<BuilderAdAccount, "id" | "name" | "currency">[];
  currency: string;
  today: string;
  disabled: boolean;
  issues: (field: string) => string[];
  brief: string;
  onBrief: (v: string) => void;
  onAi: () => void;
  aiPending: boolean;
  canAi: boolean;
};

export function SectionObjective({ state, set, adAccounts, currency, today, disabled, issues, brief, onBrief, onAi, aiPending, canAi }: Props) {
  return (
    <Block id="step-objective" index="01" title={c.sectionObjective}>
      <div className="space-y-5">
        <div className="space-y-1.5">
          <Label htmlFor="cmp-name">{c.nameLabel}</Label>
          <Input id="cmp-name" value={state.name} onChange={(e) => set("name", e.target.value)} placeholder={c.namePlaceholder} maxLength={160} disabled={disabled} />
          <FieldIssues messages={issues("name")} />
        </div>

        <fieldset className="space-y-2">
          <legend className="text-sm font-medium">{c.objectiveLabel}</legend>
          <div role="radiogroup" className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-5">
            {OBJECTIVES.map((o) => {
              const on = state.objective === o;
              return (
                <button
                  key={o}
                  type="button"
                  role="radio"
                  aria-checked={on}
                  disabled={disabled}
                  onClick={() => set("objective", o)}
                  className={cn(
                    "lift flex min-h-20 flex-col items-start gap-1 rounded-2xl bg-muted/60 p-3 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-60",
                    on && "bg-primary text-primary-foreground",
                  )}
                >
                  <span className="text-sm font-semibold">{c.objectives[o]!.label}</span>
                  <span className={cn("text-xs leading-snug", on ? "text-primary-foreground/80" : "text-muted-foreground")}>{c.objectives[o]!.desc}</span>
                </button>
              );
            })}
          </div>
        </fieldset>

        <div className="grid gap-4 sm:grid-cols-2">
          {adAccounts.length ? (
            <div className="space-y-1.5">
              <Label htmlFor="cmp-account">{c.adAccountLabel}</Label>
              <Select value={state.adAccountId ?? undefined} onValueChange={(v) => set("adAccountId", v)} disabled={disabled}>
                <SelectTrigger id="cmp-account" className="w-full">
                  <SelectValue placeholder={c.adAccountLabel} />
                </SelectTrigger>
                <SelectContent>
                  {adAccounts.map((a) => (
                    <SelectItem key={a.id} value={a.id}>
                      {a.name} · {a.currency}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FieldIssues messages={issues("adAccount")} />
            </div>
          ) : (
            <p className="self-end text-xs text-muted-foreground">{c.adAccountNone}</p>
          )}
          <div className="space-y-1.5">
            <div className="flex items-center gap-1.5">
              <Label htmlFor="cmp-budget">
                {c.budgetLabel} <span className="text-xs font-normal text-muted-foreground">({currency})</span>
              </Label>
              <InfoHint>{c.budgetMinHint(minDailyBudget(currency), currency)}</InfoHint>
            </div>
            <Input
              id="cmp-budget"
              type="number"
              inputMode="numeric"
              min={minDailyBudget(currency)}
              step={currency === "IDR" ? 1000 : 1}
              value={Number.isFinite(state.dailyBudget) && state.dailyBudget > 0 ? state.dailyBudget : ""}
              onChange={(e) => set("dailyBudget", e.target.value === "" ? 0 : Number(e.target.value))}
              disabled={disabled}
            />
            <FieldIssues messages={issues("budget")} />
          </div>
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label htmlFor="cmp-start">{c.startLabel}</Label>
            <Input id="cmp-start" type="date" min={today} value={state.startDate} onChange={(e) => set("startDate", e.target.value)} disabled={disabled} />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="cmp-end">
              {c.endLabel} <span className="text-xs font-normal text-muted-foreground">({c.endOptional})</span>
            </Label>
            <Input id="cmp-end" type="date" min={state.startDate || today} value={state.endDate} onChange={(e) => set("endDate", e.target.value)} disabled={disabled} />
          </div>
          <div className="sm:col-span-2">
            <FieldIssues messages={issues("schedule")} />
          </div>
        </div>

        {canAi ? (
          <div className="stage-tint space-y-2 rounded-2xl p-4">
            <div className="flex items-center gap-1.5">
              <Label htmlFor="cmp-brief">{c.briefLabel}</Label>
              <InfoHint>{c.briefHint}</InfoHint>
            </div>
            <Textarea id="cmp-brief" value={brief} onChange={(e) => onBrief(e.target.value)} placeholder={c.briefPlaceholder} maxLength={1000} rows={2} disabled={disabled} />
            <div className="flex justify-end">
              <Button type="button" size="sm" onClick={onAi} disabled={disabled || aiPending}>
                {aiPending ? <Loader2 className="animate-spin" /> : <Sparkles />}
                {aiPending ? c.aiFilling : c.aiFill}
              </Button>
            </div>
          </div>
        ) : null}
      </div>
    </Block>
  );
}
