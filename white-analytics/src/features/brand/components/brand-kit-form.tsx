"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Loader2, Plus } from "lucide-react";
import { toast } from "@/lib/toast";
import { InfoHint } from "@/components/dashboard/info-hint";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { saveBrandKit } from "@/features/brand/actions";
import {
  BRAND_DEFAULTS,
  BRAND_LIMITS,
  formatHashtags,
  joinSampleCaptions,
  normalizeHex,
  parseHashtagList,
  parseKeywords,
  parseLines,
  parseSampleCaptions,
} from "@/features/brand/lib";
import type { BrandKitData } from "@/features/brand/queries";
import { b } from "@/features/brand/strings";
import { cn } from "@/lib/utils";
import { BrandCard } from "./brand-card";
import { ColorField } from "./color-field";

type FormState = {
  primaryColor: string;
  secondaryColor: string;
  accentColors: string[];
  headingFont: string;
  bodyFont: string;
  logoUrl: string;
  toneOfVoice: string;
  audience: string;
  visualStyle: string;
  doList: string;
  dontList: string;
  keywords: string;
  hashtags: string;
  sampleCaptions: string;
  notes: string;
};

function toForm(k: BrandKitData | null): FormState {
  return {
    primaryColor: k?.primaryColor ?? BRAND_DEFAULTS.primaryColor,
    secondaryColor: k?.secondaryColor ?? BRAND_DEFAULTS.secondaryColor,
    accentColors: k?.accentColors ?? [],
    headingFont: k?.headingFont ?? "",
    bodyFont: k?.bodyFont ?? "",
    logoUrl: k?.logoUrl ?? "",
    toneOfVoice: k?.toneOfVoice ?? "",
    audience: k?.audience ?? "",
    visualStyle: k?.visualStyle ?? "",
    doList: (k?.doList ?? []).join("\n"),
    dontList: (k?.dontList ?? []).join("\n"),
    keywords: (k?.keywords ?? []).join(", "),
    hashtags: formatHashtags(k?.hashtags ?? []),
    sampleCaptions: joinSampleCaptions(k?.sampleCaptions ?? []),
    notes: k?.notes ?? "",
  };
}

function Block({ title, hint, children }: { title: string; hint: string; children: React.ReactNode }) {
  return (
    <section className="rounded-[1.5rem] bg-card p-6 shadow-(--card-shadow)">
      <h2 className="mb-4 flex items-center gap-1.5 text-base font-semibold tracking-[-0.01em]">
        {title} <InfoHint>{hint}</InfoHint>
      </h2>
      <div className="grid gap-4">{children}</div>
    </section>
  );
}

function F({ id, label, hint, className, children }: { id: string; label: string; hint?: string; className?: string; children: React.ReactNode }) {
  return (
    <div className={cn("min-w-0 space-y-1.5", className)}>
      <Label htmlFor={id} className="flex items-center gap-1.5">
        {label}
        {hint ? <InfoHint>{hint}</InfoHint> : null}
      </Label>
      {children}
    </div>
  );
}

export function BrandKitForm({
  slug,
  clientName,
  initial,
  canManage,
}: {
  slug: string;
  clientName: string;
  initial: BrandKitData | null;
  canManage: boolean;
}) {
  const router = useRouter();
  const [form, setForm] = React.useState<FormState>(() => toForm(initial));
  const [pending, start] = React.useTransition();
  const disabled = !canManage || pending;
  const set = <K extends keyof FormState>(key: K, value: FormState[K]) => setForm((f) => ({ ...f, [key]: value }));

  const colorsValid =
    Boolean(normalizeHex(form.primaryColor)) &&
    Boolean(normalizeHex(form.secondaryColor)) &&
    form.accentColors.every((c) => normalizeHex(c));

  const hashtagList = React.useMemo(() => parseHashtagList(form.hashtags), [form.hashtags]);

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!colorsValid) {
      toast.error(b.invalidHex);
      return;
    }
    start(async () => {
      const res = await saveBrandKit(slug, {
        primaryColor: form.primaryColor,
        secondaryColor: form.secondaryColor,
        accentColors: form.accentColors,
        headingFont: form.headingFont,
        bodyFont: form.bodyFont,
        logoUrl: form.logoUrl,
        toneOfVoice: form.toneOfVoice,
        audience: form.audience,
        visualStyle: form.visualStyle,
        doList: parseLines(form.doList),
        dontList: parseLines(form.dontList),
        keywords: parseKeywords(form.keywords),
        hashtags: hashtagList,
        sampleCaptions: parseSampleCaptions(form.sampleCaptions),
        notes: form.notes,
      });
      if (res.ok) {
        toast.success(b.saved);
        router.refresh();
      } else toast.error(res.error);
    });
  };

  return (
    <form onSubmit={submit} className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_22rem] lg:items-start">
      <div className="order-2 grid min-w-0 gap-5 lg:order-1">
        <Block title={b.sectionVisual} hint={b.sectionVisualHint}>
          <div className="grid gap-4 sm:grid-cols-2">
            <ColorField id="bk-primary" label={b.primaryColor} value={form.primaryColor} onChange={(v) => set("primaryColor", v)} disabled={disabled} />
            <ColorField id="bk-secondary" label={b.secondaryColor} value={form.secondaryColor} onChange={(v) => set("secondaryColor", v)} disabled={disabled} />
            {form.accentColors.map((c, i) => (
              <ColorField
                key={i}
                id={`bk-accent-${i}`}
                label={`${b.accentColors} ${i + 1}`}
                value={c}
                onChange={(v) => set("accentColors", form.accentColors.map((x, j) => (j === i ? v : x)))}
                onRemove={() => set("accentColors", form.accentColors.filter((_, j) => j !== i))}
                disabled={disabled}
              />
            ))}
          </div>
          {canManage && form.accentColors.length < BRAND_LIMITS.accentColors ? (
            <div>
              <Button type="button" variant="secondary" size="sm" onClick={() => set("accentColors", [...form.accentColors, "#f59e0b"])} disabled={disabled}>
                <Plus /> {b.addAccent}
              </Button>
            </div>
          ) : null}
          <div className="grid gap-4 sm:grid-cols-2">
            <F id="bk-heading" label={b.headingFont}>
              <Input id="bk-heading" value={form.headingFont} onChange={(e) => set("headingFont", e.target.value)} placeholder={b.fontPlaceholder} maxLength={BRAND_LIMITS.font} disabled={disabled} />
            </F>
            <F id="bk-body" label={b.bodyFont}>
              <Input id="bk-body" value={form.bodyFont} onChange={(e) => set("bodyFont", e.target.value)} placeholder={b.fontPlaceholder} maxLength={BRAND_LIMITS.font} disabled={disabled} />
            </F>
          </div>
          <F id="bk-logo" label={b.logoUrl} hint={b.logoUrlHint}>
            <Input id="bk-logo" type="url" inputMode="url" value={form.logoUrl} onChange={(e) => set("logoUrl", e.target.value)} placeholder="https://" disabled={disabled} />
          </F>
          <F id="bk-visual" label={b.visualStyle}>
            <Textarea id="bk-visual" rows={2} value={form.visualStyle} onChange={(e) => set("visualStyle", e.target.value)} placeholder={b.visualStylePlaceholder} maxLength={BRAND_LIMITS.text} disabled={disabled} />
          </F>
        </Block>

        <Block title={b.sectionVoice} hint={b.sectionVoiceHint}>
          <F id="bk-tone" label={b.toneOfVoice}>
            <Textarea id="bk-tone" rows={2} value={form.toneOfVoice} onChange={(e) => set("toneOfVoice", e.target.value)} placeholder={b.tonePlaceholder} maxLength={BRAND_LIMITS.text} disabled={disabled} />
          </F>
          <F id="bk-audience" label={b.audience}>
            <Textarea id="bk-audience" rows={2} value={form.audience} onChange={(e) => set("audience", e.target.value)} placeholder={b.audiencePlaceholder} maxLength={BRAND_LIMITS.text} disabled={disabled} />
          </F>
          <div className="grid gap-4 sm:grid-cols-2">
            <F id="bk-do" label={b.doList} hint={b.listHint}>
              <Textarea id="bk-do" rows={5} value={form.doList} onChange={(e) => set("doList", e.target.value)} disabled={disabled} />
            </F>
            <F id="bk-dont" label={b.dontList} hint={b.listHint}>
              <Textarea id="bk-dont" rows={5} value={form.dontList} onChange={(e) => set("dontList", e.target.value)} disabled={disabled} />
            </F>
          </div>
        </Block>

        <Block title={b.sectionLibrary} hint={b.sectionLibraryHint}>
          <F id="bk-keywords" label={b.keywords} hint={b.keywordsHint}>
            <Input id="bk-keywords" value={form.keywords} onChange={(e) => set("keywords", e.target.value)} disabled={disabled} />
          </F>
          <F id="bk-tags" label={b.hashtags} hint={b.hashtagsHint}>
            <Input id="bk-tags" value={form.hashtags} onChange={(e) => set("hashtags", e.target.value)} disabled={disabled} />
          </F>
          <F id="bk-samples" label={b.sampleCaptions} hint={b.sampleCaptionsHint}>
            <Textarea id="bk-samples" rows={7} value={form.sampleCaptions} onChange={(e) => set("sampleCaptions", e.target.value)} disabled={disabled} className="leading-relaxed" />
          </F>
          <F id="bk-notes" label={b.notes}>
            <Textarea id="bk-notes" rows={3} value={form.notes} onChange={(e) => set("notes", e.target.value)} maxLength={BRAND_LIMITS.notes} disabled={disabled} />
          </F>
        </Block>

        {canManage ? (
          <div className="flex justify-end">
            <Button type="submit" size="lg" disabled={pending || !colorsValid} className="w-full sm:w-auto">
              {pending ? <Loader2 className="animate-spin" /> : null} {b.save}
            </Button>
          </div>
        ) : null}
      </div>

      <aside className="order-1 min-w-0 lg:sticky lg:top-24 lg:order-2">
        <BrandCard
          brand={{
            clientName,
            primaryColor: form.primaryColor,
            secondaryColor: form.secondaryColor,
            accentColors: form.accentColors,
            headingFont: form.headingFont,
            bodyFont: form.bodyFont,
            logoUrl: form.logoUrl,
            toneOfVoice: form.toneOfVoice,
            hashtags: hashtagList,
          }}
        />
      </aside>
    </form>
  );
}
