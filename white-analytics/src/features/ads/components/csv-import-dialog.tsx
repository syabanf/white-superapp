"use client";

import * as React from "react";
import { Download, FileSpreadsheet, Upload, X } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { importAdsCsv } from "@/features/ads/actions";
import { parseAdsCsv, type CsvField, type ParseAdsCsvResult } from "@/features/ads/csv-import";
import { s } from "@/features/ads/strings";
import { t } from "@/i18n/id";
import { formatNumber } from "@/lib/format";
import { cn } from "@/lib/utils";

const MAX_FILE_BYTES = 10 * 1024 * 1024;

/** Kolom yang ditampilkan pada ringkasan deteksi (urut tampil). */
const PREVIEW_FIELDS: { field: CsvField; label: string }[] = [
  { field: "campaignName", label: "Nama kampanye" },
  { field: "adSetName", label: "Nama set iklan" },
  { field: "adName", label: "Nama iklan" },
  { field: "day", label: "Tanggal" },
  { field: "spend", label: "Belanja" },
  { field: "impressions", label: "Impresi" },
  { field: "reach", label: "Jangkauan" },
  { field: "linkClicks", label: "Klik tautan" },
  { field: "clicks", label: "Klik (semua)" },
  { field: "results", label: "Hasil" },
  { field: "resultType", label: "Indikator hasil" },
  { field: "age", label: "Usia" },
  { field: "gender", label: "Gender" },
  { field: "purchaseValue", label: "Nilai pembelian" },
];

const SAMPLE_CSV = [
  "Campaign name,Ad set name,Ad name,Day,Amount spent (IDR),Impressions,Reach,Link clicks,Clicks (all),Results,Result indicator,Age,Gender",
  '"[Prospecting] Kopi – Traffic","Jaksel 25-34","Video 15s",2026-08-01,350000,50000,30000,950,1200,950,actions:link_click,,',
  '"[Prospecting] Kopi – Traffic","Jaksel 25-34","Video 15s",2026-08-02,275500,41000,26000,801,1002,801,actions:link_click,,',
  '"[Prospecting] Kopi – Traffic","Jaksel 25-34","Video 15s",2026-08-01,120000,18000,12000,340,410,340,actions:link_click,25-34,male',
  '"[Prospecting] Kopi – Traffic","Jaksel 25-34","Video 15s",2026-08-01,230000,32000,18000,610,790,610,actions:link_click,25-34,female',
].join("\n");

type Preview = { result: ParseAdsCsvResult; totalPreviewRows: number };

/** Dialog impor CSV ekspor Meta Ads Manager: drop zone → pratinjau → unggah. */
export function CsvImportDialog({ clientId }: { clientId: string }) {
  const [open, setOpen] = React.useState(false);
  const [file, setFile] = React.useState<File | null>(null);
  const [preview, setPreview] = React.useState<Preview | null>(null);
  const [localError, setLocalError] = React.useState<string | null>(null);
  const [isDragging, setDragging] = React.useState(false);
  const [isPending, startTransition] = React.useTransition();
  const inputRef = React.useRef<HTMLInputElement>(null);

  const reset = () => {
    setFile(null);
    setPreview(null);
    setLocalError(null);
  };

  const acceptFile = async (f: File | undefined) => {
    setLocalError(null);
    setPreview(null);
    if (!f) return;
    if (!/\.csv$/i.test(f.name)) {
      setLocalError(s.notCsv);
      return;
    }
    if (f.size > MAX_FILE_BYTES) {
      setLocalError(s.fileTooLarge);
      return;
    }
    setFile(f);
    try {
      // Pratinjau client-side: cukup ~60 baris pertama, parsing penuh terjadi di server.
      const text = await f.text();
      const lines = text.split(/\r?\n/);
      const head = lines.slice(0, 61).join("\n");
      setPreview({ result: parseAdsCsv(head), totalPreviewRows: Math.max(0, lines.length - 1) });
    } catch {
      setLocalError(t.errors.UNKNOWN);
    }
  };

  const submit = () => {
    if (!file) {
      setLocalError(s.noFile);
      return;
    }
    const fd = new FormData();
    fd.append("file", file);
    startTransition(async () => {
      const res = await importAdsCsv(clientId, fd);
      if (res.ok) {
        toast.success(t.ads.importSuccess, { description: s.importSummary(res.data) });
        for (const w of res.data.warnings.slice(0, 2)) toast.warning(w);
        setOpen(false);
        reset();
      } else {
        toast.error(t.ads.importFailed, { description: res.error });
      }
    });
  };

  const downloadSample = () => {
    const blob = new Blob(["﻿" + SAMPLE_CSV], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "contoh-ekspor-meta-ads.csv";
    a.click();
    URL.revokeObjectURL(url);
  };

  const previewRows = preview?.result.rows.slice(0, 5) ?? [];

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) reset();
      }}
    >
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <Upload className="size-4" /> {t.ads.importCsv}
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t.ads.importCsv}</DialogTitle>
          <DialogDescription>{t.ads.importCsvDesc}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Drop zone */}
          <div
            role="button"
            tabIndex={0}
            aria-label={t.ads.dropHere}
            onClick={() => inputRef.current?.click()}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                inputRef.current?.click();
              }
            }}
            onDragOver={(e) => {
              e.preventDefault();
              setDragging(true);
            }}
            onDragLeave={() => setDragging(false)}
            onDrop={(e) => {
              e.preventDefault();
              setDragging(false);
              void acceptFile(e.dataTransfer.files?.[0]);
            }}
            className={cn(
              "flex w-full cursor-pointer flex-col items-center justify-center gap-2 rounded-xl border border-dashed px-4 py-8 text-sm text-muted-foreground transition-colors hover:bg-muted/40 focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none",
              isDragging && "border-ring bg-muted/50",
            )}
          >
            <FileSpreadsheet className="size-6" />
            {file ? (
              <span className="flex items-center gap-2 text-foreground">
                <span className="font-medium">{file.name}</span>
                <span className="text-xs text-muted-foreground">{formatNumber(Math.ceil(file.size / 1024))} KB</span>
                <button
                  type="button"
                  aria-label={s.removeFile}
                  className="rounded p-0.5 hover:bg-muted"
                  onClick={(e) => {
                    e.stopPropagation();
                    reset();
                  }}
                >
                  <X className="size-3.5" />
                </button>
              </span>
            ) : (
              <span>{t.ads.dropHere}</span>
            )}
            <input
              ref={inputRef}
              type="file"
              accept=".csv,text/csv"
              className="sr-only"
              tabIndex={-1}
              onChange={(e) => void acceptFile(e.target.files?.[0])}
            />
          </div>

          {localError ? <p className="text-sm text-destructive">{localError}</p> : null}

          {/* Pratinjau deteksi kolom */}
          {preview ? (
            <div className="space-y-3">
              <div>
                <p className="mb-1.5 text-xs font-medium text-muted-foreground">{t.ads.importDetected}</p>
                <div className="flex flex-wrap gap-1.5">
                  {PREVIEW_FIELDS.map(({ field, label }) => {
                    const hit = preview.result.detected[field];
                    return (
                      <Badge key={field} variant={hit ? "secondary" : "outline"} className={cn("font-normal", !hit && "text-muted-foreground/60")}>
                        {label}
                        {hit ? <span className="ml-1 opacity-70">← {hit}</span> : <span className="ml-1 opacity-60">· {s.detectedNone}</span>}
                      </Badge>
                    );
                  })}
                </div>
              </div>

              {preview.result.warnings.length > 0 ? (
                <ul className="list-disc space-y-0.5 pl-4 text-xs text-amber-600 dark:text-amber-500">
                  {preview.result.warnings.map((w, i) => (
                    <li key={i}>{w}</li>
                  ))}
                </ul>
              ) : null}

              {previewRows.length > 0 ? (
                <div>
                  <p className="mb-1.5 text-xs font-medium text-muted-foreground">
                    {s.previewTitle} · ±{formatNumber(preview.totalPreviewRows)} {t.ads.importRows}
                  </p>
                  <div className="overflow-x-auto rounded-lg border">
                    <Table className="[&_td]:py-1.5 [&_th]:h-8">
                      <TableHeader>
                        <TableRow className="hover:bg-transparent">
                          <TableHead className="text-xs">{t.ads.campaign}</TableHead>
                          <TableHead className="text-xs">{t.ads.ad}</TableHead>
                          <TableHead className="text-xs">{t.common.date}</TableHead>
                          <TableHead className="text-right text-xs">{t.ads.spend}</TableHead>
                          <TableHead className="text-right text-xs">{t.ads.results}</TableHead>
                          <TableHead className="text-xs">{s.rowKindLabel}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {previewRows.map((r, i) => (
                          <TableRow key={i}>
                            <TableCell className="max-w-44 truncate text-xs">{r.campaignName}</TableCell>
                            <TableCell className="max-w-36 truncate text-xs">{r.adName ?? "–"}</TableCell>
                            <TableCell className="text-xs tabular">{r.date ?? "–"}</TableCell>
                            <TableCell className="text-right text-xs tabular">{formatNumber(r.spend)}</TableCell>
                            <TableCell className="text-right text-xs tabular">{formatNumber(r.results)}</TableCell>
                            <TableCell className="text-xs">
                              {r.kind === "demographic" ? `${t.ads.demographics} (${r.age ?? "?"} · ${r.gender ?? "?"})` : s.rowKindDaily}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </div>
              ) : null}
            </div>
          ) : (
            <div className="rounded-lg bg-muted/40 p-3 text-xs text-muted-foreground">
              <p className="mb-1 flex items-center justify-between gap-2 font-medium text-foreground">
                {s.sampleFormatTitle}
                <Button type="button" variant="ghost" size="xs" onClick={downloadSample}>
                  <Download className="size-3" /> {t.common.download}
                </Button>
              </p>
              <p>{s.sampleFormatDesc}</p>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)} disabled={isPending}>
            {t.common.cancel}
          </Button>
          <Button onClick={submit} disabled={!file || isPending}>
            <Upload className="size-4" /> {isPending ? s.importing : s.uploadAndImport}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
