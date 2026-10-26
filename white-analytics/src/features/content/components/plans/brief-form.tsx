"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { PlatformIcon, platformLabel } from "@/features/social/components/platform-icon";
import { updatePlan } from "@/features/content/actions";
import { PLATFORMS, splitList } from "@/features/content/lib";
import { c } from "@/features/content/strings";
import type { Plan, PlanBrief, Platform } from "@/features/content/types";
import { Field } from "../fields";

/** The plan brief. Editable while the plan is a draft and the caller may write; read-only otherwise. */
export function BriefForm({ slug, plan, editable }: { slug: string; plan: Plan; editable: boolean }) {
  const router = useRouter();
  const brief = plan.brief;
  const [title, setTitle] = React.useState(plan.title);
  const [goals, setGoals] = React.useState(brief?.goals ?? "");
  const [pillars, setPillars] = React.useState((brief?.pillars ?? []).join(", "));
  const [tone, setTone] = React.useState(brief?.tone ?? "");
  const [notes, setNotes] = React.useState(brief?.notes ?? "");
  const [freq, setFreq] = React.useState<Record<Platform, number>>(() => {
    const out = { INSTAGRAM: 0, FACEBOOK: 0, TIKTOK: 0 } as Record<Platform, number>;
    for (const f of brief?.frequency ?? []) out[f.platform] = f.postsPerWeek;
    return out;
  });
  const [pending, start] = React.useTransition();

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const next: PlanBrief = {
      goals: goals.trim(),
      pillars: splitList(pillars),
      frequency: PLATFORMS.filter((p) => freq[p] > 0).map((p) => ({ platform: p, postsPerWeek: freq[p] })),
      tone: tone.trim(),
      notes: notes.trim(),
    };
    start(async () => {
      const res = await updatePlan({ slug, planId: plan.id, title: title.trim() || undefined, brief: next });
      if (res.ok) {
        toast.success(c.briefSaved);
        router.refresh();
      } else toast.error(res.error);
    });
  };

  const disabled = !editable || pending;
  return (
    <form onSubmit={submit} className="grid gap-4 rounded-[1.5rem] bg-card p-6 shadow-(--card-shadow) sm:grid-cols-2">
      <Field id="brief-title" label={c.titleLabel} className="sm:col-span-2">
        <Input id="brief-title" value={title} onChange={(e) => setTitle(e.target.value)} disabled={disabled} />
      </Field>
      <Field id="brief-goals" label={c.goals} className="sm:col-span-2">
        <Textarea id="brief-goals" rows={2} value={goals} onChange={(e) => setGoals(e.target.value)} placeholder={c.goalsPlaceholder} disabled={disabled} />
      </Field>
      <Field id="brief-pillars" label={c.pillars} hint={c.pillarsHint} className="sm:col-span-2">
        <Input id="brief-pillars" value={pillars} onChange={(e) => setPillars(e.target.value)} disabled={disabled} />
      </Field>
      <fieldset className="space-y-1.5 sm:col-span-2">
        <legend className="text-sm font-medium">{c.frequency}</legend>
        <div className="grid grid-cols-3 gap-3">
          {PLATFORMS.map((p) => (
            <label key={p} className="flex items-center gap-2 rounded-lg border px-3 py-1.5 text-sm">
              <PlatformIcon platform={p} className="size-4" />
              <span className="hidden min-w-0 flex-1 truncate sm:inline">{platformLabel(p)}</span>
              <Input
                type="number"
                min={0}
                max={21}
                value={freq[p]}
                onChange={(e) => setFreq((s) => ({ ...s, [p]: Math.max(0, Number(e.target.value) || 0) }))}
                className="h-7 w-16 text-right tabular"
                aria-label={platformLabel(p)}
                disabled={disabled}
              />
            </label>
          ))}
        </div>
      </fieldset>
      <Field id="brief-tone" label={c.tone}>
        <Input id="brief-tone" value={tone} onChange={(e) => setTone(e.target.value)} placeholder={c.tonePlaceholder} disabled={disabled} />
      </Field>
      <Field id="brief-notes" label={c.notes}>
        <Input id="brief-notes" value={notes} onChange={(e) => setNotes(e.target.value)} disabled={disabled} />
      </Field>
      {editable ? (
        <div className="flex justify-end sm:col-span-2">
          <Button type="submit" size="sm" disabled={pending}>
            {pending ? <Loader2 className="size-3.5 animate-spin" /> : null} {c.saveBrief}
          </Button>
        </div>
      ) : null}
    </form>
  );
}
