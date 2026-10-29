"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Loader2, Wand2 } from "lucide-react";
import { toast } from "@/lib/toast";
import { Section } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { generateDesign } from "@/features/design-studio/actions";
import { DESIGN_FORMATS, FORMAT_LABEL, formatSizeLabel, type DesignFormat, type DraftRow } from "@/features/design-studio/lib";
import { ds } from "@/features/design-studio/strings";
import { DesignResult } from "./design-result";
import { DraftHistory } from "./draft-history";

type Brief = { title: string; brief: string; format: DesignFormat };

/** Brief form → generated draft; history below re-opens earlier drafts in the same result panel. */
export function DesignStudio({ slug, drafts: initialDrafts, canManage }: { slug: string; drafts: DraftRow[]; canManage: boolean }) {
  const router = useRouter();
  const [drafts, setDrafts] = React.useState(initialDrafts);
  // Server refresh (after generate/delete) replaces the local list; the documented "adjust state on prop change" pattern.
  const [prevInitial, setPrevInitial] = React.useState(initialDrafts);
  if (prevInitial !== initialDrafts) {
    setPrevInitial(initialDrafts);
    setDrafts(initialDrafts);
  }
  const [selected, setSelected] = React.useState<DraftRow | null>(initialDrafts[0] ?? null);
  const [form, setForm] = React.useState<Brief>({ title: "", brief: "", format: "FEED_SQUARE" });
  const [pending, start] = React.useTransition();
  const resultRef = React.useRef<HTMLDivElement>(null);

  const run = (input: Brief) => {
    if (!input.title.trim()) {
      toast.error(ds.titleRequired);
      return;
    }
    start(async () => {
      const res = await generateDesign(slug, input);
      if (res.ok) {
        setSelected(res.data.draft);
        setDrafts((list) => [res.data.draft, ...list.filter((d) => d.id !== res.data.draft.id)].slice(0, 24));
        toast.success(ds.generated);
        resultRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
        router.refresh();
      } else toast.error(res.error);
    });
  };

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    run(form);
  };

  const regenerate = () => {
    if (!selected) return;
    const input = { title: selected.title, brief: selected.brief, format: selected.format };
    setForm(input);
    run(input);
  };

  const updated = (d: DraftRow) => {
    setSelected(d);
    setDrafts((list) => list.map((x) => (x.id === d.id ? d : x)));
  };

  const deleted = (id: string) => {
    setDrafts((list) => list.filter((d) => d.id !== id));
    if (selected?.id === id) setSelected(null);
    router.refresh();
  };

  return (
    <>
      {canManage ? (
        <form onSubmit={submit} className="grid gap-4 rounded-[1.5rem] bg-card p-6 shadow-(--card-shadow)">
          <h2 className="text-base font-semibold tracking-[-0.01em]">{ds.formTitle}</h2>
          <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_14rem]">
            <div className="min-w-0 space-y-1.5">
              <Label htmlFor="ds-title">{ds.titleLabel}</Label>
              <Input id="ds-title" value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} placeholder={ds.titlePlaceholder} maxLength={120} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="ds-format">{ds.formatLabel}</Label>
              <Select value={form.format} onValueChange={(v) => setForm({ ...form, format: v as DesignFormat })}>
                <SelectTrigger id="ds-format" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {DESIGN_FORMATS.map((f) => (
                    <SelectItem key={f} value={f}>
                      {FORMAT_LABEL[f]} <span className="text-muted-foreground tabular">· {formatSizeLabel(f)}</span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="ds-brief">{ds.briefLabel}</Label>
            <Textarea id="ds-brief" rows={4} value={form.brief} onChange={(e) => setForm({ ...form, brief: e.target.value })} placeholder={ds.briefPlaceholder} maxLength={2000} />
          </div>
          <div className="flex justify-end">
            <Button type="submit" disabled={pending} className="w-full sm:w-auto">
              {pending ? <Loader2 className="animate-spin" /> : <Wand2 />} {pending ? ds.generating : ds.generate}
            </Button>
          </div>
        </form>
      ) : (
        <p className="text-sm text-muted-foreground">{ds.readOnly}</p>
      )}

      <div ref={resultRef} className="scroll-mt-24">
        <Section title={ds.resultTitle}>
          {selected ? (
            <DesignResult slug={slug} draft={selected} canManage={canManage} regenerating={pending} onRegenerate={regenerate} onUpdated={updated} />
          ) : (
            <div className="flex min-h-56 flex-col items-center justify-center gap-2 rounded-[1.5rem] bg-card p-6 text-center shadow-(--card-shadow)">
              <span className="flex size-11 items-center justify-center rounded-full bg-muted text-muted-foreground">
                <Wand2 className="size-5" />
              </span>
              <p className="text-sm font-semibold">{ds.resultEmptyTitle}</p>
              <p className="max-w-sm text-xs text-muted-foreground">{ds.resultEmptyDesc}</p>
            </div>
          )}
        </Section>
      </div>

      <Section title={ds.historyTitle} description={ds.historyHint}>
        <DraftHistory drafts={drafts} selectedId={selected?.id ?? null} canManage={canManage} onSelect={setSelected} onDeleted={deleted} />
      </Section>
    </>
  );
}
