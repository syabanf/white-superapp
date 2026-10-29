"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { toast } from "@/lib/toast";
import { formatDate } from "@/lib/format";
import { deleteCampaignDraft, markCampaignReady, saveCampaignDraft, submitCampaignDraft, suggestCampaignWithAi } from "@/features/campaigns/actions";
import { draftIssues, type DraftStatus } from "@/features/campaigns/lib";
import type { BuilderAdAccount, BuilderBrand, BuilderImage, DraftDTO } from "@/features/campaigns/queries";
import type { WriterGate } from "@/lib/providers/meta-ads-writer/types";
import { c, objectiveLabel } from "@/features/campaigns/strings";
import { initialBuilderState, toDraftInput, type BuilderState, type SetField } from "./builder-state";
import { SectionObjective } from "./section-objective";
import { SectionAudience, SectionPlacements } from "./section-audience";
import { SectionCreatives } from "./section-creatives";
import { SectionSummary } from "./section-summary";
import { AdPreview } from "./ad-preview";
import { BuilderActions, type PendingAction } from "./builder-actions";

type Props = {
  clientId: string;
  slug: string;
  today: string;
  clientCurrency: string;
  adAccounts: BuilderAdAccount[];
  images: BuilderImage[];
  brand: BuilderBrand;
  noAccountGate: WriterGate;
  initial: DraftDTO | null;
  canManage: boolean;
};

const snapshot = (s: BuilderState) => JSON.stringify(s);

export function CampaignBuilder({ clientId, slug, today, clientCurrency, adAccounts, images, brand, noAccountGate, initial, canManage }: Props) {
  const router = useRouter();
  const listHref = `/clients/${slug}/ads/campaigns`;
  const [state, setState] = React.useState<BuilderState>(() =>
    initialBuilderState(initial, { today, defaultAdAccountId: adAccounts.length === 1 ? adAccounts[0]!.id : null, defaultBudget: clientCurrency === "IDR" ? 50_000 : 10 }),
  );
  const [saved, setSaved] = React.useState(() => (initial ? snapshot(state) : ""));
  const [draftId, setDraftId] = React.useState(initial?.id ?? null);
  const [status, setStatus] = React.useState<DraftStatus>(initial?.status ?? "DRAFT");
  const [submitted, setSubmitted] = React.useState(
    initial?.status === "SUBMITTED" ? { externalId: initial.externalId, simulated: initial.simulated, submittedAt: initial.submittedAt } : null,
  );
  const [brief, setBrief] = React.useState(initial?.name ?? "");
  const [attempted, setAttempted] = React.useState(false);
  const [pending, setPending] = React.useState<PendingAction>(null);
  const [aiPending, startAi] = React.useTransition();
  const [active, setActive] = React.useState(0);

  const set: SetField = React.useCallback((key, value) => setState((s) => ({ ...s, [key]: value })), []);
  const account = adAccounts.find((a) => a.id === state.adAccountId) ?? null;
  const currency = account?.currency ?? clientCurrency;
  const gate = account?.gate ?? noAccountGate;
  const readOnly = !canManage || status === "SUBMITTED";
  const dirty = snapshot(state) !== saved;
  const issues = draftIssues(toDraftInput(state, currency), { today, requireAdAccount: adAccounts.length > 0 });
  const fieldIssues = (field: string) => (attempted ? Array.from(new Set(issues.filter((i) => i.field === field).map((i) => i.message))) : []);

  /** Persist the form; returns the draft id or null on failure (already toasted). */
  const persist = async (): Promise<{ id: string; created: boolean } | null> => {
    const next = state.name.trim() ? state : { ...state, name: `${objectiveLabel(state.objective)} · ${formatDate(state.startDate)}` };
    const res = await saveCampaignDraft({ clientId, id: draftId ?? undefined, ...next, endDate: next.endDate || null });
    if (!res.ok) {
      toast.error(res.error);
      return null;
    }
    if (next !== state) setState(next);
    setSaved(snapshot(next));
    setDraftId(res.data.id);
    setStatus(res.data.status);
    return { id: res.data.id, created: res.data.created };
  };

  const afterWrite = (r: { id: string; created: boolean }) => {
    if (r.created) router.replace(`${listHref}/${r.id}`);
    else router.refresh();
  };

  const run = (action: Exclude<PendingAction, null>, fn: () => Promise<void>) => {
    setPending(action);
    fn().finally(() => setPending(null));
  };

  const onSave = () =>
    run("save", async () => {
      const r = await persist();
      if (!r) return;
      toast.success(r.created ? c.toast.created : c.toast.saved);
      afterWrite(r);
    });

  const onReady = () => {
    setAttempted(true);
    if (issues.length) {
      toast.error(issues[0]!.message);
      const f = issues[0]!.field;
      const target = f.startsWith("creative-") ? f : `step-${sectionOf(f)}`;
      document.getElementById(target)?.scrollIntoView({ behavior: "smooth", block: "center" });
      return;
    }
    run("ready", async () => {
      const r = dirty || !draftId ? await persist() : { id: draftId, created: false };
      if (!r) return;
      const res = await markCampaignReady(r.id);
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      setStatus("READY");
      toast.success(c.toast.ready);
      afterWrite(r);
    });
  };

  const onSubmit = () =>
    run("submit", async () => {
      const r = dirty || !draftId ? await persist() : { id: draftId, created: false };
      if (!r) return;
      const res = await submitCampaignDraft(r.id);
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      setStatus("SUBMITTED");
      setSubmitted({ externalId: res.data.externalId, simulated: res.data.simulated, submittedAt: new Date().toISOString() });
      toast.success(res.data.simulated ? c.toast.submittedSim : c.toast.submitted);
      document.getElementById("step-summary")?.scrollIntoView({ behavior: "smooth", block: "start" });
      afterWrite(r);
    });

  const onDelete = () =>
    run("delete", async () => {
      if (!draftId) return;
      const res = await deleteCampaignDraft(draftId);
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      toast.success(c.toast.deleted);
      router.push(listHref);
    });

  const onAi = () =>
    startAi(async () => {
      const res = await suggestCampaignWithAi({
        clientId,
        objective: state.objective,
        brief: brief.trim() || state.name,
        dailyBudget: state.dailyBudget,
        adAccountId: state.adAccountId,
        audience: state.audience,
        placements: state.placements,
        creatives: state.creatives,
      });
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      const { audience, placements, creatives, aiNotes, mock } = res.data;
      setState((s) => ({ ...s, audience, placements, creatives, aiNotes: aiNotes || s.aiNotes }));
      toast.success(mock ? c.toast.aiFilledMock : c.toast.aiFilled);
    });

  const sectionProps = { state, set, disabled: readOnly, issues: fieldIssues };
  const preview = <AdPreview brand={brand} creatives={state.creatives} active={active} onActive={setActive} />;

  return (
    <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px] xl:grid-cols-[minmax(0,1fr)_380px]">
      <div className="min-w-0 space-y-5">
        {!canManage ? <Note>{c.viewerReadOnly}</Note> : status === "SUBMITTED" ? <Note>{c.submittedReadOnly}</Note> : null}
        <SectionObjective
          {...sectionProps}
          adAccounts={adAccounts}
          currency={currency}
          today={today}
          brief={brief}
          onBrief={setBrief}
          onAi={onAi}
          aiPending={aiPending}
          canAi={!readOnly}
        />
        <SectionAudience {...sectionProps} />
        <SectionPlacements {...sectionProps} />
        <SectionCreatives {...sectionProps} images={images} active={active} onActive={setActive} />
        <div className="lg:hidden">{preview}</div>
        <SectionSummary state={state} set={set} currency={currency} gate={gate} issues={issues} disabled={readOnly} submitted={submitted} />
        {canManage ? (
          <BuilderActions
            status={status}
            canDelete={Boolean(draftId)}
            simulated={gate.mode === "simulation"}
            pending={pending}
            onSave={onSave}
            onReady={onReady}
            onSubmit={onSubmit}
            onDelete={onDelete}
          />
        ) : null}
      </div>
      <aside className="hidden min-w-0 lg:sticky lg:top-6 lg:block lg:self-start">{preview}</aside>
    </div>
  );
}

function Note({ children }: { children: React.ReactNode }) {
  return <p className="rounded-2xl border border-dashed px-4 py-2.5 text-sm text-muted-foreground">{children}</p>;
}

function sectionOf(field: string): string {
  if (field === "audience") return "audience";
  if (field === "placements") return "placements";
  if (field === "creatives") return "creatives";
  return "objective";
}
