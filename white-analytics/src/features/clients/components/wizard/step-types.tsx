"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { ArrowRight, Megaphone, Search, Share2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { WizardNav } from "@/features/clients/components/wizard/wizard-nav";
import { cn } from "@/lib/utils";
import { PROJECT_TYPES, wizardHref, type ProjectType } from "@/features/clients/wizard";
import { tc } from "@/features/clients/strings";
import { t } from "@/i18n/id";

const CHOICES: { key: ProjectType; icon: typeof Search; label: string; desc: string }[] = [
  { key: "SEO", icon: Search, label: tc.wizard.seoLabel, desc: tc.wizard.seoDesc },
  { key: "SOCIAL", icon: Share2, label: tc.wizard.socialLabel, desc: tc.wizard.socialDesc },
  { key: "ADS", icon: Megaphone, label: tc.wizard.adsLabel, desc: tc.wizard.adsDesc },
];

/**
 * Step 1 — what kind of project this is. Nothing is written yet; the choice
 * travels in the URL so the operator can go back and change it freely.
 */
export function StepTypes({ initial }: { initial: ProjectType[] }) {
  const router = useRouter();
  const [selected, setSelected] = React.useState<ProjectType[]>(initial);
  const [error, setError] = React.useState<string>();
  const [pending, startTransition] = React.useTransition();

  const toggle = (key: ProjectType) =>
    setSelected((cur) => (cur.includes(key) ? cur.filter((k) => k !== key) : [...cur, key]));

  const selectAll = () => setSelected([...PROJECT_TYPES]);
  const allSelected = selected.length === PROJECT_TYPES.length;

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    if (selected.length === 0) {
      setError(tc.wizard.pickOneChannel);
      return;
    }
    setError(undefined);
    startTransition(() => router.push(wizardHref("profil", { types: selected })));
  };

  return (
    <form onSubmit={submit} className="space-y-5">
      <div className="grid gap-3">
        {CHOICES.map((c) => {
          const on = selected.includes(c.key);
          return (
            <label
              key={c.key}
              className={cn(
                "lift flex cursor-pointer items-start gap-3.5 rounded-xl border p-4",
                on ? "border-brand/45 bg-accent/60" : "hover:bg-muted/50",
              )}
            >
              <Checkbox checked={on} onCheckedChange={() => toggle(c.key)} disabled={pending} className="mt-0.5" />
              <span className="min-w-0 flex-1">
                <span className="flex items-center gap-2 text-sm font-semibold">
                  <c.icon className={cn("size-4", on ? "text-brand" : "text-muted-foreground")} />
                  {c.label}
                </span>
                <span className="mt-1 block text-xs leading-relaxed text-muted-foreground">{c.desc}</span>
              </span>
            </label>
          );
        })}
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-xs text-muted-foreground">{tc.wizard.channelsHint}</p>
        {!allSelected ? (
          <Button type="button" variant="ghost" size="xs" onClick={selectAll} disabled={pending}>
            {tc.wizard.selectAll}
          </Button>
        ) : null}
      </div>

      {error ? (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      ) : null}

      <WizardNav
        backHref={{ href: "/clients", label: t.common.cancel }}
        submit={
          <Button type="submit" disabled={pending} className="group/nudge">
            {tc.wizard.next} <ArrowRight className="nudge nudge-x size-4" />
          </Button>
        }
      />
    </form>
  );
}
