"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Loader2, Save, SendHorizonal } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { PlatformIcon, platformLabel } from "@/features/social/components/platform-icon";
import { savePost, submitForReview, type SavePostInput } from "@/features/publishing/actions";
import { PLATFORMS, canTransition, defaultUtm, hasUtm, validatePost, type Platform, type PostRole, type Utm } from "@/features/publishing/lib";
import type { BestTimeSlot, MediaRow, OwnAccount, PostDetail } from "@/features/publishing/queries";
import { fromZoned, toZoned } from "@/features/publishing/time";
import { p } from "@/features/publishing/strings";
import { t } from "@/i18n/id";
import { TargetPicker } from "./target-picker";
import { CaptionEditor } from "./caption-editor";
import { MediaPicker } from "./media-picker";
import { ExtrasFields, LinkUtmFields } from "./link-utm";
import { SchedulePicker } from "./schedule-picker";
import { PostPreview } from "./preview";

type Props = {
  clientId: string;
  slug: string;
  timezone: string;
  today: string;
  accounts: OwnAccount[];
  library: MediaRow[];
  bestTimes: BestTimeSlot[];
  initial: PostDetail | null;
  role: PostRole;
};

function Block({ index, title, hint, children }: { index: string; title: string; hint?: string; children: React.ReactNode }) {
  return (
    <section className="rounded-[1.5rem] bg-card p-6 shadow-(--card-shadow)">
      <header className="mb-3">
        <h2 className="flex items-baseline gap-2 text-sm font-semibold tracking-[-0.01em]">
          <span className="label-mono text-brand">{index}</span> {title}
        </h2>
        {hint ? <p className="mt-0.5 text-xs text-muted-foreground">{hint}</p> : null}
      </header>
      {children}
    </section>
  );
}

export function Composer({ clientId, slug, timezone, today, accounts, library: initialLibrary, bestTimes, initial, role }: Props) {
  const router = useRouter();
  const base = `/clients/${slug}/publish`;
  const [pending, start] = React.useTransition();
  const readOnly = role === "VIEWER";

  const initialOverrides = React.useMemo(() => {
    const out: Partial<Record<Platform, string>> = {};
    for (const tg of initial?.targets ?? []) if (tg.bodyOverride && !out[tg.platform]) out[tg.platform] = tg.bodyOverride;
    return out;
  }, [initial]);
  const initialSchedule = initial?.scheduledAt ? toZoned(new Date(initial.scheduledAt), timezone) : null;

  const [library, setLibrary] = React.useState(initialLibrary);
  const [title, setTitle] = React.useState(initial?.title ?? "");
  const [body, setBody] = React.useState(initial?.body ?? "");
  const [targetIds, setTargetIds] = React.useState<string[]>(initial?.targets.map((tg) => tg.socialAccountId) ?? []);
  const [overrides, setOverrides] = React.useState(initialOverrides);
  const [media, setMedia] = React.useState<MediaRow[]>(initial?.media ?? []);
  const [alt, setAlt] = React.useState<Record<string, string>>({});
  const [linkUrl, setLinkUrl] = React.useState(initial?.linkUrl ?? "");
  const [utm, setUtm] = React.useState<Utm>(initial?.utm ?? {});
  const [firstComment, setFirstComment] = React.useState(initial?.firstComment ?? "");
  const [labels, setLabels] = React.useState<string[]>(initial?.labels ?? []);
  const [date, setDate] = React.useState(initialSchedule?.date ?? "");
  const [time, setTime] = React.useState(initialSchedule?.time ?? "");
  const [issues, setIssues] = React.useState<string[]>([]);

  const selectedAccounts = accounts.filter((a) => targetIds.includes(a.id));
  const platforms = PLATFORMS.filter((pl) => selectedAccounts.some((a) => a.platform === pl));
  const [previewPlatform, setPreviewPlatform] = React.useState<Platform>(platforms[0] ?? "INSTAGRAM");
  const activePreview = platforms.includes(previewPlatform) ? previewPlatform : platforms[0];
  const previewAccount = selectedAccounts.find((a) => a.platform === activePreview);

  const onLink = (v: string) => {
    setLinkUrl(v);
    if (v.trim() && !hasUtm(utm)) setUtm(defaultUtm(platforms[0]));
  };

  const buildPayload = (): SavePostInput => ({
    clientId,
    id: initial?.id,
    title,
    body,
    targets: targetIds.map((id) => {
      const acc = accounts.find((a) => a.id === id);
      return { socialAccountId: id, bodyOverride: acc ? overrides[acc.platform]?.trim() || null : null };
    }),
    mediaIds: media.map((m) => m.id),
    mediaAlt: alt,
    linkUrl: linkUrl.trim() || null,
    utm: linkUrl.trim() && hasUtm(utm) ? utm : undefined,
    firstComment: platforms.includes("INSTAGRAM") ? firstComment.trim() || null : null,
    labels,
    scheduledAt: date && time ? fromZoned(date, time, timezone).toISOString() : null,
  });

  const submit = (mode: "draft" | "review") => {
    const check = validatePost(
      {
        body,
        targets: targetIds.map((id) => ({ platform: accounts.find((a) => a.id === id)?.platform ?? "INSTAGRAM", bodyOverride: overrides[accounts.find((a) => a.id === id)?.platform ?? "INSTAGRAM"] })),
        media: media.map((m) => ({ kind: m.kind })),
        linkUrl,
        firstComment,
      },
      { level: mode === "review" ? "full" : "draft" },
    );
    if (!check.ok) {
      const msgs = Array.from(new Set(check.issues.map((i) => (i.scope === "post" ? i.message : `${platformLabel(i.scope)}: ${i.message}`))));
      setIssues(msgs);
      toast.error(msgs[0]!);
      return;
    }
    setIssues([]);
    start(async () => {
      const saved = await savePost(buildPayload());
      if (!saved.ok) {
        toast.error(saved.error);
        return;
      }
      if (mode === "review") {
        const sub = await submitForReview(saved.data.id);
        if (!sub.ok) {
          toast.error(sub.error);
          router.push(`${base}/posts/${saved.data.id}`);
          return;
        }
        toast.success(p.toast.submitted);
      } else toast.success(initial ? p.savedChanges : p.savedDraft);
      router.push(`${base}/posts/${saved.data.id}`);
    });
  };

  const canSubmitReview = !readOnly && canTransition(initial?.status ?? "DRAFT", "IN_REVIEW", { role, isAuthor: true });
  const disabled = pending || readOnly;

  return (
    <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px] xl:grid-cols-[minmax(0,1fr)_380px]">
      <div className="min-w-0 space-y-4">
        {readOnly ? <p className="rounded-lg border border-dashed px-3 py-2 text-sm text-muted-foreground">{p.viewerReadOnly}</p> : null}
        {initial && initial.status !== "DRAFT" ? <p className="rounded-lg border border-dashed px-3 py-2 text-xs text-muted-foreground">{p.editResetsNote}</p> : null}

        <Block index="01" title={p.sectionTargets}>
          <TargetPicker accounts={accounts} selected={targetIds} onChange={setTargetIds} settingsHref={`/clients/${slug}/settings`} disabled={disabled} />
        </Block>

        <Block index="02" title={p.sectionCaption}>
          <div className="mb-3 space-y-1">
            <Label htmlFor="post-title">
              {p.titleLabel} <span className="text-xs font-normal text-muted-foreground">({t.common.optional})</span>
            </Label>
            <Input id="post-title" value={title} onChange={(e) => setTitle(e.target.value)} placeholder={p.titlePlaceholder} disabled={disabled} className="h-9" maxLength={160} />
          </div>
          <CaptionEditor body={body} onBody={setBody} platforms={platforms} overrides={overrides} onOverride={(pl, v) => setOverrides((o) => ({ ...o, [pl]: v }))} disabled={disabled} />
        </Block>

        <Block index="03" title={p.sectionMedia}>
          <MediaPicker
            clientId={clientId}
            library={library}
            onLibraryAdd={(assets) => setLibrary((cur) => [...assets, ...cur])}
            selected={media}
            onChange={setMedia}
            alt={alt}
            onAlt={(id, v) => setAlt((a) => ({ ...a, [id]: v }))}
            disabled={disabled}
          />
        </Block>

        <Block index="04" title={p.sectionLink}>
          <LinkUtmFields linkUrl={linkUrl} onLink={onLink} utm={utm} onUtm={setUtm} disabled={disabled} />
        </Block>

        <Block index="05" title={p.sectionExtras}>
          <ExtrasFields showFirstComment={platforms.includes("INSTAGRAM")} firstComment={firstComment} onFirstComment={setFirstComment} labels={labels} onLabels={setLabels} disabled={disabled} />
        </Block>

        <Block index="06" title={p.sectionSchedule} hint={p.timezoneNote(timezone)}>
          <SchedulePicker
            date={date}
            time={time}
            min={today}
            onChange={(d, tm) => {
              setDate(d);
              setTime(tm);
            }}
            timeZone={timezone}
            bestTimes={bestTimes}
            disabled={disabled}
          />
        </Block>

        {issues.length ? (
          <ul role="alert" className="space-y-1 rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
            {issues.map((m) => (
              <li key={m}>{m}</li>
            ))}
          </ul>
        ) : null}

        <div className="sticky bottom-0 z-10 -mx-4 flex items-center justify-end gap-2 border-t bg-background/95 px-4 py-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] backdrop-blur md:-mx-6 md:px-6 lg:static lg:mx-0 lg:border-0 lg:bg-transparent lg:p-0 lg:backdrop-blur-none">
          <Button type="button" variant="outline" size="sm" onClick={() => submit("draft")} disabled={disabled}>
            {pending ? <Loader2 className="size-3.5 animate-spin" /> : <Save className="size-3.5" />}
            {pending ? p.saving : initial ? p.saveChanges : p.saveDraft}
          </Button>
          {canSubmitReview ? (
            <Button type="button" size="sm" onClick={() => submit("review")} disabled={disabled}>
              <SendHorizonal className="size-3.5" /> {p.saveAndSubmit}
            </Button>
          ) : null}
        </div>
      </div>

      <aside className="min-w-0 lg:sticky lg:top-6 lg:self-start">
        <div className="rounded-[1.5rem] bg-card p-6 shadow-(--card-shadow)">
          <div className="mb-3 flex items-center justify-between gap-2">
            <h2 className="flex items-baseline gap-2 text-sm font-semibold">
              <span className="label-mono text-brand">07</span> {p.sectionPreview}
            </h2>
            {platforms.length > 1 ? (
              <Tabs value={activePreview} onValueChange={(v) => setPreviewPlatform(v as Platform)}>
                <TabsList className="h-8">
                  {platforms.map((pl) => (
                    <TabsTrigger key={pl} value={pl} aria-label={platformLabel(pl)}>
                      <PlatformIcon platform={pl} className="size-3.5" mono={activePreview !== pl} />
                    </TabsTrigger>
                  ))}
                </TabsList>
              </Tabs>
            ) : null}
          </div>
          {activePreview && previewAccount ? (
            <PostPreview
              platform={activePreview}
              account={previewAccount}
              caption={overrides[activePreview]?.trim() ? overrides[activePreview]! : body}
              media={media}
              linkUrl={linkUrl.trim() || undefined}
            />
          ) : (
            <p className="py-10 text-center text-sm text-muted-foreground">{p.previewEmpty}</p>
          )}
        </div>
      </aside>
    </div>
  );
}
