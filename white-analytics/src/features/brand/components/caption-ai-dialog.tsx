"use client";

import * as React from "react";
import { Loader2, RefreshCw, Sparkles } from "lucide-react";
import { toast } from "@/lib/toast";
import { InfoHint } from "@/components/dashboard/info-hint";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { rewriteCaptionWithAi } from "@/features/brand/actions";
import { formatHashtags } from "@/features/brand/lib";
import { b } from "@/features/brand/strings";
import type { CaptionVariant } from "@/lib/providers/ai-studio/types";

export type CaptionPlatform = "INSTAGRAM" | "FACEBOOK" | "TIKTOK";

const PLATFORM_LABEL: Record<CaptionPlatform, string> = { INSTAGRAM: "Instagram", FACEBOOK: "Facebook", TIKTOK: "TikTok" };

/**
 * "AI" button + dialog: sends the editor's current hook/caption/hashtags and an
 * optional instruction, shows 3 variants, and hands the chosen one back via
 * `onApply`. It never saves; the host editor keeps its own save flow.
 */
export function CaptionAiDialog({
  slug,
  platform: initialPlatform = "INSTAGRAM",
  read,
  onApply,
  disabled,
}: {
  slug: string;
  platform?: CaptionPlatform;
  /** current editor values, read when the request is sent */
  read: () => { hook: string; caption: string; hashtags: string[] };
  onApply: (v: CaptionVariant) => void;
  disabled?: boolean;
}) {
  const [open, setOpen] = React.useState(false);
  const [instruction, setInstruction] = React.useState("");
  const [platform, setPlatform] = React.useState<CaptionPlatform>(initialPlatform);
  const [variants, setVariants] = React.useState<CaptionVariant[]>([]);
  const [demo, setDemo] = React.useState(false);
  const [pending, start] = React.useTransition();

  const run = () => {
    const current = read();
    if (!current.caption.trim() && !current.hook.trim()) {
      toast.error(b.emptyInput);
      return;
    }
    start(async () => {
      const res = await rewriteCaptionWithAi({ slug, ...current, instruction, platform });
      if (res.ok) {
        setVariants(res.data.variants);
        setDemo(res.data.demo);
      } else toast.error(res.error);
    });
  };

  const apply = (v: CaptionVariant) => {
    onApply(v);
    setOpen(false);
    toast.success(b.applied);
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button type="button" variant="secondary" size="xs" disabled={disabled}>
          <Sparkles /> {b.aiButton}
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-1.5">
            {b.aiTitle} <InfoHint>{b.aiHint}</InfoHint>
          </DialogTitle>
        </DialogHeader>

        <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_10rem]">
          <div className="min-w-0 space-y-1.5">
            <Label htmlFor="cap-ai-instruction">{b.instruction}</Label>
            <Input
              id="cap-ai-instruction"
              value={instruction}
              onChange={(e) => setInstruction(e.target.value)}
              placeholder={b.instructionPlaceholder}
              maxLength={500}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  run();
                }
              }}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="cap-ai-platform">{b.platform}</Label>
            <Select value={platform} onValueChange={(v) => setPlatform(v as CaptionPlatform)}>
              <SelectTrigger id="cap-ai-platform" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {(Object.keys(PLATFORM_LABEL) as CaptionPlatform[]).map((p) => (
                  <SelectItem key={p} value={p}>
                    {PLATFORM_LABEL[p]}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-2">
          {demo ? <span className="text-xs text-muted-foreground">{b.demoNote}</span> : <span />}
          <Button type="button" size="sm" onClick={run} disabled={pending}>
            {pending ? <Loader2 className="animate-spin" /> : variants.length ? <RefreshCw /> : <Sparkles />}
            {variants.length ? b.regenerate : b.generate}
          </Button>
        </div>

        {variants.length ? (
          <ol className="grid gap-3" aria-busy={pending}>
            {variants.map((v, i) => (
              <li key={`${v.label}-${i}`} className="rounded-[1.25rem] bg-muted/50 p-4">
                <div className="mb-2 flex items-center justify-between gap-2">
                  <span className="label-mono text-brand">{v.label}</span>
                  <Button type="button" size="xs" onClick={() => apply(v)}>
                    {b.use}
                  </Button>
                </div>
                {v.hook ? <p className="text-sm font-semibold">{v.hook}</p> : null}
                <p className="mt-1 text-[13px] leading-relaxed whitespace-pre-wrap">{v.caption}</p>
                {v.hashtags.length ? <p className="mt-2 text-xs break-words text-brand-ink">{formatHashtags(v.hashtags)}</p> : null}
              </li>
            ))}
          </ol>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
