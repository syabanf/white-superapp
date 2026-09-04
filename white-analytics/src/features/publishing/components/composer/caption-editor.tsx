"use client";

import { ChevronDown } from "lucide-react";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { PlatformIcon, platformLabel } from "@/features/social/components/platform-icon";
import { countHashtags, PLATFORM_LIMITS, type Platform } from "@/features/publishing/lib";
import { p } from "@/features/publishing/strings";
import { formatNumber } from "@/lib/format";
import { cn } from "@/lib/utils";

function Counter({ platform, text }: { platform: Platform; text: string }) {
  const lim = PLATFORM_LIMITS[platform];
  const over = text.length > lim.caption;
  const tags = countHashtags(text);
  const tagsOver = lim.hashtags != null && tags > lim.hashtags;
  return (
    <span className={cn("inline-flex items-center gap-1 text-xs tabular", over || tagsOver ? "text-destructive" : "text-muted-foreground")}>
      <PlatformIcon platform={platform} className="size-3" mono />
      {formatNumber(text.length)}/{formatNumber(lim.caption)}
      {lim.hashtags != null ? (
        <span className="ml-1">
          · {tags}/{lim.hashtags} {p.hashtags}
        </span>
      ) : null}
    </span>
  );
}

export function CaptionEditor({
  body,
  onBody,
  platforms,
  overrides,
  onOverride,
  disabled,
}: {
  body: string;
  onBody: (v: string) => void;
  /** platforms of the selected targets (unique) */
  platforms: Platform[];
  overrides: Partial<Record<Platform, string>>;
  onOverride: (platform: Platform, v: string) => void;
  disabled?: boolean;
}) {
  return (
    <div className="space-y-3">
      <Textarea
        value={body}
        onChange={(e) => onBody(e.target.value)}
        placeholder={p.captionPlaceholder}
        rows={7}
        disabled={disabled}
        className="min-h-40 resize-y text-sm leading-relaxed"
      />
      <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
        {(platforms.length ? platforms : (["INSTAGRAM"] as Platform[])).map((pl) => (
          <Counter key={pl} platform={pl} text={overrides[pl]?.trim() ? overrides[pl]! : body} />
        ))}
      </div>
      {platforms.length > 1 || Object.values(overrides).some(Boolean) ? (
        <div className="space-y-1.5">
          {platforms.map((pl) => {
            const value = overrides[pl] ?? "";
            return (
              <Collapsible key={pl} defaultOpen={Boolean(value)}>
                <CollapsibleTrigger asChild>
                  <button
                    type="button"
                    className="group/ov flex w-full items-center justify-between rounded-lg border px-3 py-2 text-left text-[13px] font-medium transition-colors duration-150 hover:bg-muted/50"
                  >
                    <span className="inline-flex items-center gap-2">
                      <PlatformIcon platform={pl} className="size-3.5" /> {p.overrideFor(platformLabel(pl))}
                      {value.trim() ? <span className="label-mono text-brand">aktif</span> : null}
                    </span>
                    <ChevronDown className="size-4 text-muted-foreground transition-transform duration-200 group-data-[state=open]/ov:rotate-180" />
                  </button>
                </CollapsibleTrigger>
                <CollapsibleContent className="pt-2">
                  <Textarea
                    value={value}
                    onChange={(e) => onOverride(pl, e.target.value)}
                    placeholder={p.overridePlaceholder}
                    rows={4}
                    disabled={disabled}
                    className="text-sm"
                  />
                  <div className="mt-1.5 flex items-center justify-between">
                    <Counter platform={pl} text={value.trim() ? value : body} />
                    {value ? (
                      <Button type="button" variant="ghost" size="xs" onClick={() => onOverride(pl, "")} disabled={disabled}>
                        {p.useMaster}
                      </Button>
                    ) : null}
                  </div>
                </CollapsibleContent>
              </Collapsible>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
