"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { proxyRequest } from "@/features/content/client";
import { formatBytes } from "@/features/content/lib";
import { c } from "@/features/content/strings";
import { Field } from "./fields";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  url: string;
  /** Multipart field for the files: `files` (repeated) for design versions, `file` for assets. */
  fileField: "files" | "file";
  multiple?: boolean;
  accept?: string;
  success: string;
  /** Extra form fields rendered above the file input; their values come from `extra()`. */
  children?: React.ReactNode;
  extra?: () => Record<string, string> | string;
  noteLabel?: string;
  onDone?: (body: unknown) => void;
};

/** Multipart upload through the gateway proxy. `extra` may return a string to block submit with that message. */
export function UploadDialog({
  open,
  onOpenChange,
  title,
  description,
  url,
  fileField,
  multiple,
  accept,
  success,
  children,
  extra,
  noteLabel,
  onDone,
}: Props) {
  const router = useRouter();
  const [files, setFiles] = React.useState<File[]>([]);
  const [note, setNote] = React.useState("");
  const [busy, setBusy] = React.useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (files.length === 0) return;
    const fields = extra?.() ?? {};
    if (typeof fields === "string") {
      toast.error(fields);
      return;
    }
    const form = new FormData();
    for (const f of files) form.append(fileField, f);
    if (noteLabel) form.set("note", note);
    for (const [k, v] of Object.entries(fields)) form.set(k, v);
    setBusy(true);
    const res = await proxyRequest<unknown>(url, { method: "POST", body: form });
    setBusy(false);
    if (!res.ok) {
      toast.error(`${c.uploadFailed}: ${res.error}`);
      return;
    }
    toast.success(success);
    setFiles([]);
    setNote("");
    onOpenChange(false);
    if (onDone) onDone(res.data);
    else router.refresh();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <form id="upload-form" onSubmit={submit} className="space-y-4">
          {children}
          <Field id="up-files" label={c.files} hint={files.length ? c.filesPicked(files.length) : undefined}>
            <Input id="up-files" type="file" multiple={multiple} accept={accept} onChange={(e) => setFiles(Array.from(e.target.files ?? []))} />
          </Field>
          {files.length > 1 ? (
            <ol className="list-decimal space-y-0.5 pl-5 text-xs text-muted-foreground">
              {files.map((f) => (
                <li key={`${f.name}-${f.size}`}>
                  {f.name} · {formatBytes(f.size)}
                </li>
              ))}
            </ol>
          ) : null}
          {noteLabel ? (
            <Field id="up-note" label={noteLabel}>
              <Input id="up-note" value={note} onChange={(e) => setNote(e.target.value)} />
            </Field>
          ) : null}
        </form>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {c.cancel}
          </Button>
          <Button type="submit" form="upload-form" disabled={busy || files.length === 0}>
            {busy ? <Loader2 className="size-3.5 animate-spin" /> : null} {title}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
