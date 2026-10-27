"use client";

import * as React from "react";
import { Check, CircleDashed, FileIcon, FileUp, Loader2, SkipForward } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { formatDateTime } from "@/lib/format";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { fileSize } from "../lib";
import { ps } from "../strings";
import type { PublicOnboarding } from "../types";
import { fileToBase64, publicClientFetch } from "./client";
import { FieldInput } from "./field-input";

const o = ps.onboarding;
const MAX_BYTES = 15 * 1024 * 1024;

/** Client portal: progress, brief form and asset uploads behind the onboarding token. */
export function OnboardingPortal({ token, initial }: { token: string; initial: PublicOnboarding }) {
  const [data, setData] = React.useState(initial);
  const base = `/v1/public/sales/onboarding/${encodeURIComponent(token)}`;

  return (
    <div className="page-stack">
      <Card className="gap-3 p-(--card-pad)">
        <div className="flex items-center justify-between gap-3 text-sm">
          <span className="font-medium">{o.progress}</span>
          <span className="font-semibold tabular">{data.progress}%</span>
        </div>
        <Progress value={data.progress} />
        <ol className="mt-2 grid gap-2 sm:grid-cols-2">
          {data.steps.map((st, i) => (
            <li key={i} className="flex items-center gap-2 text-sm">
              <span
                className={cn(
                  "flex size-5 shrink-0 items-center justify-center rounded-full border",
                  st.status === "done" && "border-positive bg-positive text-white",
                )}
              >
                {st.status === "done" ? (
                  <Check className="size-3" />
                ) : st.status === "skipped" ? (
                  <SkipForward className="size-3 text-muted-foreground" />
                ) : (
                  <CircleDashed className="size-3 text-muted-foreground" />
                )}
              </span>
              <span className={cn(st.status !== "todo" && "text-muted-foreground")}>{st.title}</span>
              <span className="sr-only">{o.stepStatus[st.status] ?? st.status}</span>
            </li>
          ))}
        </ol>
      </Card>
      <div className="grid gap-(--gap) lg:grid-cols-[1fr_340px]">
        <BriefForm data={data} base={base} onSaved={setData} />
        <Assets data={data} base={base} onUploaded={setData} />
      </div>
    </div>
  );
}

function BriefForm({ data, base, onSaved }: { data: PublicOnboarding; base: string; onSaved: (d: PublicOnboarding) => void }) {
  const [values, setValues] = React.useState<Record<string, string>>(data.brief ?? {});
  const [errors, setErrors] = React.useState<Record<string, string>>({});
  const [pending, setPending] = React.useState(false);
  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    const missing = Object.fromEntries(data.briefFields.filter((f) => f.required && !values[f.key]?.trim()).map((f) => [f.key, ps.required]));
    setErrors(missing);
    if (Object.keys(missing).length) return;
    setPending(true);
    try {
      const res = await publicClientFetch<PublicOnboarding>(`${base}/brief`, { method: "POST", body: JSON.stringify({ brief: values }) });
      onSaved(res);
      toast.success(o.briefSaved);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : ps.offline);
    } finally {
      setPending(false);
    }
  };
  return (
    <Card className="p-(--card-pad)">
      <form className="flex flex-col gap-4" onSubmit={submit} noValidate>
        <div>
          <h2 className="text-base font-semibold">{o.brief}</h2>
          {data.briefSubmittedAt ? <p className="text-xs text-muted-foreground">{o.briefSubmitted(formatDateTime(data.briefSubmittedAt))}</p> : null}
        </div>
        {data.briefFields.map((f) => (
          <FieldInput key={f.key} field={f} value={values[f.key] ?? ""} onChange={(v) => setValues((p) => ({ ...p, [f.key]: v }))} error={errors[f.key]} />
        ))}
        <Button type="submit" disabled={pending} className="self-start">
          {pending ? <Loader2 className="size-4 animate-spin" /> : null}
          {o.briefSave}
        </Button>
      </form>
    </Card>
  );
}

function Assets({ data, base, onUploaded }: { data: PublicOnboarding; base: string; onUploaded: (d: PublicOnboarding) => void }) {
  const input = React.useRef<HTMLInputElement>(null);
  const [busy, setBusy] = React.useState<string | null>(null);
  const upload = async (files: FileList | null) => {
    for (const file of Array.from(files ?? [])) {
      if (file.size > MAX_BYTES) {
        toast.error(`${file.name}: ${o.tooLarge}`);
        continue;
      }
      setBusy(file.name);
      try {
        const res = await publicClientFetch<PublicOnboarding>(`${base}/assets`, {
          method: "POST",
          body: JSON.stringify({ fileName: file.name, data: await fileToBase64(file) }),
        });
        onUploaded(res);
        toast.success(`${o.uploaded}: ${file.name}`);
      } catch (err) {
        toast.error(`${file.name}: ${err instanceof Error ? err.message : ps.offline}`);
      }
    }
    setBusy(null);
  };
  return (
    <Card className="h-fit gap-3 p-(--card-pad)">
      <div>
        <h2 className="text-base font-semibold">{o.assets}</h2>
        <p className="text-xs text-muted-foreground">{o.assetsHint}</p>
      </div>
      <Button type="button" variant="outline" disabled={!!busy} onClick={() => input.current?.click()}>
        {busy ? <Loader2 className="size-4 animate-spin" /> : <FileUp className="size-4" />}
        {busy ? o.uploading(busy) : o.upload}
      </Button>
      <input
        ref={input}
        type="file"
        hidden
        multiple
        accept="image/png,image/jpeg,image/webp,image/gif,image/svg+xml,application/pdf,application/zip,.zip"
        onChange={(e) => {
          void upload(e.target.files);
          e.target.value = "";
        }}
      />
      {data.assets.length ? (
        <ul className="divide-y text-sm">
          {data.assets.map((a) => (
            <li key={a.id} className="flex items-center gap-2 py-2">
              <FileIcon className="size-4 shrink-0 text-muted-foreground" />
              <span className="min-w-0 flex-1 truncate">{a.fileName}</span>
              <span className="text-xs text-muted-foreground">{fileSize(a.sizeBytes)}</span>
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-sm text-muted-foreground">{o.noAssets}</p>
      )}
    </Card>
  );
}
