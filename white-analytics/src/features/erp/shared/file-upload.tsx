"use client";

import * as React from "react";
import { CheckCircle2, FileUp, Loader2, XCircle } from "lucide-react";
import { Progress } from "@/components/ui/progress";
import { cn } from "@/lib/utils";
import { ss } from "./strings";

type Item = { key: string; name: string; progress: number; state: "uploading" | "done" | "error"; error?: string };

/** POSTs one file as multipart (field `file`) with upload progress; resolves with the JSON body. */
export function uploadWithProgress<T>(
  url: string,
  file: File,
  extra: Record<string, string>,
  onProgress: (pct: number) => void,
): Promise<T> {
  return new Promise((resolve, reject) => {
    const form = new FormData();
    form.append("file", file);
    for (const [k, v] of Object.entries(extra)) form.append(k, v);
    const xhr = new XMLHttpRequest();
    xhr.open("POST", url);
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(Math.round((e.loaded / e.total) * 100));
    };
    xhr.onload = () => {
      let body: unknown = null;
      try {
        body = JSON.parse(xhr.responseText);
      } catch {
        body = null;
      }
      if (xhr.status >= 200 && xhr.status < 300) resolve(body as T);
      else reject(new Error((body as { error?: string } | null)?.error ?? ss.upload.failed));
    };
    xhr.onerror = () => reject(new Error(ss.upload.failed));
    xhr.send(form);
  });
}

/**
 * Dropzone that uploads straight to a proxy route, e.g.
 * `/api/erp/api/finance/reimbursements/{id}/receipts`.
 */
export function FileUpload<T = unknown>({
  url,
  accept = "image/*,application/pdf",
  acceptLabel = "JPG, PNG, PDF",
  maxMb = 10,
  multiple = true,
  extraFields = {},
  disabled,
  onUploaded,
  className,
}: {
  url: string;
  accept?: string;
  acceptLabel?: string;
  maxMb?: number;
  multiple?: boolean;
  extraFields?: Record<string, string>;
  disabled?: boolean;
  onUploaded?: (body: T, file: File) => void;
  className?: string;
}) {
  const inputRef = React.useRef<HTMLInputElement>(null);
  const [items, setItems] = React.useState<Item[]>([]);
  const [dragging, setDragging] = React.useState(false);

  const patch = (key: string, p: Partial<Item>) =>
    setItems((list) => list.map((it) => (it.key === key ? { ...it, ...p } : it)));

  const accepts = (file: File) =>
    accept
      .split(",")
      .map((a) => a.trim())
      .some((a) => (a.endsWith("/*") ? file.type.startsWith(a.slice(0, -1)) : a.startsWith(".") ? file.name.toLowerCase().endsWith(a) : file.type === a));

  const start = (files: FileList | null) => {
    if (!files || disabled) return;
    for (const file of Array.from(files).slice(0, multiple ? undefined : 1)) {
      const key = `${file.name}-${file.size}-${Date.now()}-${Math.random()}`;
      const error = !accepts(file) ? ss.upload.wrongType : file.size > maxMb * 1024 * 1024 ? ss.upload.tooLarge(maxMb) : undefined;
      setItems((list) => [...list, { key, name: file.name, progress: 0, state: error ? "error" : "uploading", error }]);
      if (error) continue;
      uploadWithProgress<T>(url, file, extraFields, (progress) => patch(key, { progress }))
        .then((body) => {
          patch(key, { state: "done", progress: 100 });
          onUploaded?.(body, file);
          setTimeout(() => setItems((list) => list.filter((it) => it.key !== key)), 2500);
        })
        .catch((e: Error) => patch(key, { state: "error", error: e.message }));
    }
  };

  return (
    <div className={cn("flex flex-col gap-2", className)}>
      <button
        type="button"
        disabled={disabled}
        onClick={() => inputRef.current?.click()}
        onDragOver={(e) => {
          e.preventDefault();
          setDragging(true);
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={(e) => {
          e.preventDefault();
          setDragging(false);
          start(e.dataTransfer.files);
        }}
        className={cn(
          "flex flex-col items-center justify-center gap-1.5 rounded-xl border border-dashed bg-card/50 px-4 py-6 text-center text-sm transition-colors outline-none hover:bg-accent/50 focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50",
          dragging && "border-brand bg-brand/5",
        )}
      >
        <FileUp className="size-5 text-muted-foreground" />
        <span className="font-medium">{ss.upload.drop}</span>
        <span className="text-xs text-muted-foreground">{ss.upload.hint(acceptLabel, maxMb)}</span>
      </button>
      <input
        ref={inputRef}
        type="file"
        hidden
        accept={accept}
        multiple={multiple}
        onChange={(e) => {
          start(e.target.files);
          e.target.value = "";
        }}
      />
      {items.length ? (
        <ul className="flex flex-col gap-1.5">
          {items.map((it) => (
            <li key={it.key} className="rounded-lg border px-3 py-2 text-xs">
              <div className="flex items-center gap-2">
                {it.state === "uploading" ? (
                  <Loader2 className="size-3.5 animate-spin text-muted-foreground" />
                ) : it.state === "done" ? (
                  <CheckCircle2 className="size-3.5 text-positive" />
                ) : (
                  <XCircle className="size-3.5 text-destructive" />
                )}
                <span className="min-w-0 flex-1 truncate">{it.name}</span>
                <span className="text-muted-foreground tabular">
                  {it.state === "uploading" ? `${it.progress}%` : it.state === "done" ? ss.upload.done : null}
                </span>
              </div>
              {it.state === "uploading" ? <Progress value={it.progress} className="mt-1.5" /> : null}
              {it.error ? <p className="mt-1 text-destructive">{it.error}</p> : null}
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
