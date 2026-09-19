"use client";

import * as React from "react";
import { X } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { buildUtmUrl, type Utm } from "@/features/publishing/lib";
import { p } from "@/features/publishing/strings";

export function LinkUtmFields({
  linkUrl,
  onLink,
  utm,
  onUtm,
  disabled,
}: {
  linkUrl: string;
  onLink: (v: string) => void;
  utm: Utm;
  onUtm: (v: Utm) => void;
  disabled?: boolean;
}) {
  const finalUrl = buildUtmUrl(linkUrl, utm);
  const field = (key: keyof Utm, label: string) => (
    <div className="space-y-1">
      <Label htmlFor={`utm-${key}`} className="label-mono text-muted-foreground">
        {label}
      </Label>
      <Input
        id={`utm-${key}`}
        value={utm[key] ?? ""}
        onChange={(e) => onUtm({ ...utm, [key]: e.target.value })}
        disabled={disabled}
        className="h-8 font-mono text-xs"
      />
    </div>
  );
  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <Label htmlFor="post-link">{p.linkLabel}</Label>
        <Input
          id="post-link"
          type="url"
          inputMode="url"
          value={linkUrl}
          onChange={(e) => onLink(e.target.value)}
          placeholder={p.linkPlaceholder}
          disabled={disabled}
          className="h-9"
        />
      </div>
      {linkUrl.trim() ? (
        <>
          <div className="grid gap-2 sm:grid-cols-3">
            {field("source", p.utmSource)}
            {field("medium", p.utmMedium)}
            {field("campaign", p.utmCampaign)}
          </div>
          <div>
            <p className="label-mono mb-1 text-muted-foreground">{p.utmPreview}</p>
            <p className="break-all rounded-md bg-muted/60 px-2.5 py-1.5 font-mono text-xs leading-relaxed">
              {finalUrl}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">{p.utmHint}</p>
          </div>
        </>
      ) : null}
    </div>
  );
}

export function ExtrasFields({
  showFirstComment,
  firstComment,
  onFirstComment,
  labels,
  onLabels,
  disabled,
}: {
  showFirstComment: boolean;
  firstComment: string;
  onFirstComment: (v: string) => void;
  labels: string[];
  onLabels: (v: string[]) => void;
  disabled?: boolean;
}) {
  const [draft, setDraft] = React.useState("");
  const add = () => {
    const v = draft.trim().replace(/,+$/, "");
    if (v && !labels.includes(v)) onLabels([...labels, v]);
    setDraft("");
  };
  return (
    <div className="space-y-4">
      {showFirstComment ? (
        <div className="space-y-1">
          <Label htmlFor="post-first-comment">{p.firstComment}</Label>
          <Textarea
            id="post-first-comment"
            value={firstComment}
            onChange={(e) => onFirstComment(e.target.value)}
            placeholder={p.firstCommentPlaceholder}
            rows={2}
            disabled={disabled}
            className="text-sm"
          />
        </div>
      ) : null}
      <div className="space-y-1.5">
        <Label htmlFor="post-labels">{p.labels}</Label>
        <div className="flex flex-wrap items-center gap-1.5">
          {labels.map((l) => (
            <Badge key={l} variant="secondary" className="gap-1 pr-1">
              {l}
              <button
                type="button"
                onClick={() => onLabels(labels.filter((x) => x !== l))}
                aria-label={`${p.remove} ${l}`}
                className="rounded-full hover:bg-foreground/10"
                disabled={disabled}
              >
                <X className="size-3" />
              </button>
            </Badge>
          ))}
          <Input
            id="post-labels"
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === ",") {
                e.preventDefault();
                add();
              }
            }}
            onBlur={add}
            placeholder={p.labelsPlaceholder}
            disabled={disabled}
            className="h-8 w-48 text-xs"
          />
        </div>
      </div>
    </div>
  );
}
