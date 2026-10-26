"use client";

import * as React from "react";
import { FileSpreadsheet, Loader2, Upload } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { importAccountsAction } from "../actions";
import type { ImportAccountsResponse } from "../types";
import { fs } from "../strings";

const im = fs.accounts.import;
const MAX_BYTES = 5 * 1024 * 1024;

function toBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result).split(",")[1] ?? "");
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(file);
  });
}

/** Two-step XLSX import: preview (server validates every row), then commit. */
export function CoaImportDialog() {
  const [open, setOpen] = React.useState(false);
  const [file, setFile] = React.useState<{ name: string; data: string } | null>(null);
  const [preview, setPreview] = React.useState<ImportAccountsResponse | null>(null);
  const [error, setError] = React.useState("");
  const inputRef = React.useRef<HTMLInputElement>(null);
  const { pending, run } = useErpAction();

  const reset = () => {
    setFile(null);
    setPreview(null);
    setError("");
  };

  const pick = async (f: File | undefined) => {
    reset();
    if (!f) return;
    if (f.size > MAX_BYTES) return setError(im.tooLarge);
    const data = await toBase64(f);
    setFile({ name: f.name, data });
    void run(() => importAccountsAction(data, "preview"), { onOk: setPreview });
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (!o) reset();
      }}
    >
      <DialogTrigger asChild>
        <Button size="sm" variant="outline">
          <FileSpreadsheet className="size-4" /> {im.button}
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>{im.title}</DialogTitle>
          <DialogDescription>{im.desc}</DialogDescription>
        </DialogHeader>
        <div className="flex flex-wrap items-center gap-3">
          <Button variant="outline" onClick={() => inputRef.current?.click()} disabled={pending}>
            {pending && !preview ? <Loader2 className="size-4 animate-spin" /> : <Upload className="size-4" />}
            {im.pick}
          </Button>
          {file ? <span className="truncate text-sm text-muted-foreground">{file.name}</span> : null}
          <input
            ref={inputRef}
            type="file"
            hidden
            accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
            onChange={(e) => {
              void pick(e.target.files?.[0]);
              e.target.value = "";
            }}
          />
        </div>
        {error ? <p className="text-sm text-destructive">{error}</p> : null}
        {preview ? (
          <div className="flex flex-col gap-3">
            <p className="text-sm font-medium">
              {im.counts(preview.createCount, preview.updateCount, preview.skipCount, preview.errorCount)}
            </p>
            {preview.message ? <p className="text-xs text-muted-foreground">{preview.message}</p> : null}
            {preview.issues.length ? (
              <Alert variant="destructive" className="rounded-xl">
                <AlertDescription>
                  <ul className="list-disc pl-4 text-xs">
                    {preview.issues.slice(0, 8).map((i, idx) => (
                      <li key={idx}>
                        {im.row} {i.row}: {i.message}
                      </li>
                    ))}
                  </ul>
                </AlertDescription>
              </Alert>
            ) : null}
            <div className="max-h-80 overflow-auto rounded-lg border scrollbar-thin">
              <table className="w-full text-xs">
                <thead className="sticky top-0 bg-card text-muted-foreground">
                  <tr>
                    <th className="px-2 py-1.5 text-left font-medium">{im.row}</th>
                    <th className="px-2 py-1.5 text-left font-medium">{fs.code}</th>
                    <th className="px-2 py-1.5 text-left font-medium">{fs.name}</th>
                    <th className="px-2 py-1.5 text-left font-medium">{im.parentCode}</th>
                    <th className="px-2 py-1.5 text-left font-medium">{im.typeCode}</th>
                    <th className="px-2 py-1.5 text-left font-medium">{im.action}</th>
                    <th className="px-2 py-1.5 text-left font-medium">{im.message}</th>
                  </tr>
                </thead>
                <tbody>
                  {preview.preview.map((r) => (
                    <tr key={r.sourceRow} className="border-t">
                      <td className="px-2 py-1.5 tabular">{r.sourceRow}</td>
                      <td className="px-2 py-1.5 font-mono">{r.code}</td>
                      <td className="px-2 py-1.5">{r.name}</td>
                      <td className="px-2 py-1.5 font-mono">{r.parentCode}</td>
                      <td className="px-2 py-1.5">{r.accountTypeCode}</td>
                      <td className="px-2 py-1.5">
                        <ErpStatusBadge
                          status={r.action.toUpperCase()}
                          labels={Object.fromEntries(Object.entries(im.actions).map(([k, v]) => [k.toUpperCase(), v]))}
                          kinds={{ CREATE: "good", UPDATE: "info", SKIP: "neutral", ERROR: "critical" }}
                        />
                      </td>
                      <td className="px-2 py-1.5 text-muted-foreground">{r.message}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {preview.errorCount > 0 ? <p className="text-xs text-destructive">{im.hasErrors}</p> : null}
          </div>
        ) : null}
        <DialogFooter>
          <Button variant="ghost" onClick={() => setOpen(false)} disabled={pending}>
            {fs.cancel}
          </Button>
          <Button
            disabled={!file || !preview || preview.errorCount > 0 || pending}
            onClick={() =>
              file &&
              run(() => importAccountsAction(file.data, "commit"), {
                success: im.done,
                onOk: () => setOpen(false),
              })
            }
          >
            {pending && preview ? <Loader2 className="size-4 animate-spin" /> : null}
            {pending && preview ? im.committing : im.commit}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
