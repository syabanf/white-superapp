"use client";

import { DownloadButton } from "@/features/erp/shared/download-button";
import { fs } from "../strings";

const s = fs.taxSetup;

/** CSV exports of the tax register for the period on screen. */
export function TaxExportButtons({ from, to }: { from?: string; to?: string }) {
  const period = new URLSearchParams({ ...(from ? { dateFrom: from } : {}), ...(to ? { dateTo: to } : {}) });
  return (
    <div className="flex flex-col gap-2 print:hidden">
      <div className="flex flex-wrap gap-2">
        {Object.entries(s.exports).map(([kind, label]) => (
          <DownloadButton key={kind} href={`/api/erp/v1/finance/reports/tax-export?kind=${kind}&${period}`} fileName={`${kind}.csv`}>
            {label}
          </DownloadButton>
        ))}
      </div>
      <p className="text-xs text-muted-foreground">{s.exportHint}</p>
    </div>
  );
}
