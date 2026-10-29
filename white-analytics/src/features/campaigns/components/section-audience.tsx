"use client";

import { Check, Wand2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { AGE_MAX, AGE_MIN, MAX_INTERESTS, MAX_LOCATIONS, OBJECTIVE_DEFAULTS, PLACEMENTS, placementPlatform, type Audience, type Placement } from "@/features/campaigns/lib";
import { c } from "@/features/campaigns/strings";
import type { BuilderState, SetField } from "./builder-state";
import { Block, FieldIssues } from "./block";
import { TagInput } from "./tag-input";

type SectionProps = { state: BuilderState; set: SetField; disabled: boolean; issues: (field: string) => string[] };

const GENDERS: Audience["genders"][] = ["ALL", "MALE", "FEMALE"];
const AGES = Array.from({ length: AGE_MAX - AGE_MIN + 1 }, (_, i) => AGE_MIN + i);

function AgeSelect({ label, value, onChange, disabled }: { label: string; value: number; onChange: (v: number) => void; disabled: boolean }) {
  return (
    <Select value={String(value)} onValueChange={(v) => onChange(Number(v))} disabled={disabled}>
      <SelectTrigger aria-label={label} className="w-24">
        <SelectValue />
      </SelectTrigger>
      <SelectContent className="max-h-64">
        {AGES.map((n) => (
          <SelectItem key={n} value={String(n)}>
            {n === AGE_MAX ? `${n}+` : n}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

export function SectionAudience({ state, set, disabled, issues }: SectionProps) {
  const a = state.audience;
  const patch = (next: Partial<Audience>) => set("audience", { ...a, ...next });
  return (
    <Block id="step-audience" index="02" title={c.sectionAudience}>
      <div className="space-y-5">
        <div className="space-y-1.5">
          <Label htmlFor="cmp-locations">{c.locationsLabel}</Label>
          <TagInput id="cmp-locations" values={a.locations} onChange={(v) => patch({ locations: v })} placeholder={c.locationsPlaceholder} max={MAX_LOCATIONS} disabled={disabled} />
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <fieldset className="space-y-1.5">
            <legend className="mb-1.5 text-sm font-medium">{c.ageLabel}</legend>
            <div className="flex items-center gap-2">
              <AgeSelect label={`${c.ageLabel} min`} value={a.ageMin} onChange={(v) => patch({ ageMin: v, ageMax: Math.max(v, a.ageMax) })} disabled={disabled} />
              <span className="text-sm text-muted-foreground">{c.ageTo}</span>
              <AgeSelect label={`${c.ageLabel} max`} value={a.ageMax} onChange={(v) => patch({ ageMax: v, ageMin: Math.min(v, a.ageMin) })} disabled={disabled} />
            </div>
          </fieldset>
          <fieldset>
            <legend className="mb-1.5 text-sm font-medium">{c.genderLabel}</legend>
            <div role="radiogroup" className="inline-flex rounded-full bg-muted p-1">
              {GENDERS.map((g) => (
                <button
                  key={g}
                  type="button"
                  role="radio"
                  aria-checked={a.genders === g}
                  disabled={disabled}
                  onClick={() => patch({ genders: g })}
                  className={cn(
                    "rounded-full px-3.5 py-1.5 text-sm font-medium text-muted-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-60",
                    a.genders === g && "bg-card text-foreground shadow-(--card-shadow)",
                  )}
                >
                  {c.genders[g]}
                </button>
              ))}
            </div>
          </fieldset>
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="cmp-interests">{c.interestsLabel}</Label>
          <TagInput id="cmp-interests" values={a.interests} onChange={(v) => patch({ interests: v })} placeholder={c.interestsPlaceholder} max={MAX_INTERESTS} disabled={disabled} />
        </div>
        <FieldIssues messages={issues("audience")} />
      </div>
    </Block>
  );
}

const GROUPS = ["instagram", "facebook", "audience_network"] as const;

export function SectionPlacements({ state, set, disabled, issues }: SectionProps) {
  const toggle = (p: Placement) => set("placements", state.placements.includes(p) ? state.placements.filter((x) => x !== p) : [...state.placements, p]);
  const recommended = OBJECTIVE_DEFAULTS[state.objective].placements;
  return (
    <Block
      id="step-placements"
      index="03"
      title={c.sectionPlacements}
      actions={
        <>
          {!disabled ? (
            <Button type="button" variant="outline" size="xs" onClick={() => set("placements", [...recommended])}>
              <Wand2 /> {c.objectives[state.objective]?.label}
            </Button>
          ) : null}
        </>
      }
    >
      <div className="grid gap-4 sm:grid-cols-3">
        {GROUPS.map((g) => (
          <fieldset key={g} className="min-w-0">
            <legend className="label-mono mb-2 text-muted-foreground">{c.platformGroups[g]}</legend>
            <div className="flex flex-wrap gap-1.5">
              {PLACEMENTS.filter((p) => placementPlatform(p) === g).map((p) => {
                const on = state.placements.includes(p);
                return (
                  <button
                    key={p}
                    type="button"
                    role="checkbox"
                    aria-checked={on}
                    disabled={disabled}
                    onClick={() => toggle(p)}
                    className={cn(
                      "inline-flex items-center gap-1.5 rounded-full bg-muted px-3 py-1.5 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-60",
                      on && "bg-primary text-primary-foreground",
                    )}
                  >
                    {on ? <Check className="size-3.5" /> : null}
                    {c.placements[p]}
                  </button>
                );
              })}
            </div>
          </fieldset>
        ))}
      </div>
      {!state.placements.length ? <p className="mt-3 text-xs text-muted-foreground">{c.placementsAuto}</p> : null}
      <FieldIssues messages={issues("placements")} />
    </Block>
  );
}
