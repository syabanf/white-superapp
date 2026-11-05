"use client";

import * as React from "react";
import Link from "next/link";
import { ChevronDown, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { UrlPagination, type ServerPage } from "@/features/erp/shared/data-table";
import { Money } from "@/features/erp/shared/money-display";
import { DateRangeFilter } from "@/features/erp/shared/period-picker";
import { ResetFilters, SearchFilter, SearchSelectFilter, SelectFilter } from "@/features/erp/shared/url-filters";
import { formatDate } from "@/lib/format";
import { fs } from "../strings";
import type { JournalEntry, PostableAccount } from "../types";

const j = fs.journals;

/** Posted journals with their lines one click away, for tracing an amount back to its entry. */
export function JournalHistory({ entries, page, accounts }: { entries: JournalEntry[]; page: ServerPage; accounts: PostableAccount[] }) {
  const [open, setOpen] = React.useState<ReadonlySet<string>>(new Set());
  const toggle = (id: string) =>
    setOpen((prev) => {
      const next = new Set(prev);
      if (!next.delete(id)) next.add(id);
      return next;
    });

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <SearchFilter placeholder={j.searchPh} />
        <SelectFilter
          param="type"
          label={fs.type}
          allLabel={j.allTypes}
          options={Object.entries(fs.entryType).map(([value, label]) => ({ value, label }))}
        />
        <SearchSelectFilter
          param="account"
          placeholder={j.allAccounts}
          options={accounts.map((a) => ({ value: a.id, label: `${a.code} ${a.name}` }))}
        />
        <DateRangeFilter />
        <ResetFilters params={["q", "type", "account", "from", "to"]} />
      </div>

      {entries.length === 0 ? (
        <Card className="p-(--card-pad) text-sm text-muted-foreground">{j.historyEmpty}</Card>
      ) : (
        <Card className="gap-0 divide-y overflow-hidden py-0">
          {entries.map((e) => {
            const expanded = open.has(e.id);
            return (
              <div key={e.id}>
                <div className="flex items-center gap-3 px-3 py-2.5 text-sm">
                  <Button
                    size="icon-xs"
                    variant="ghost"
                    aria-expanded={expanded}
                    aria-label={expanded ? j.collapse : j.expand}
                    onClick={() => toggle(e.id)}
                  >
                    {expanded ? <ChevronDown className="size-4" /> : <ChevronRight className="size-4" />}
                  </Button>
                  <Link href={`/erp/finance/journals/${e.id}`} className="shrink-0 font-mono text-xs font-medium hover:underline">
                    {e.entryNo || "–"}
                  </Link>
                  <span className="shrink-0 text-xs text-muted-foreground tabular">{formatDate(e.entryDate)}</span>
                  <span className="min-w-0 flex-1 truncate">{e.description || fs.noDescription}</span>
                  <span className="hidden shrink-0 text-xs text-muted-foreground md:inline">
                    {fs.entryType[e.entryType] ?? e.entryType}
                    {e.sourceModule ? ` · ${e.sourceModule}` : ""}
                  </span>
                  <Money value={e.totalDebit} className="shrink-0 font-medium" />
                </div>
                {expanded ? (
                  <div className="overflow-x-auto border-t bg-muted/40">
                    <table className="w-full text-sm">
                      <thead className="text-xs text-muted-foreground">
                        <tr>
                          <th className="px-4 py-2 text-left font-medium">{fs.account}</th>
                          <th className="px-4 py-2 text-left font-medium">{fs.description}</th>
                          <th className="px-4 py-2 text-right font-medium">{j.debit}</th>
                          <th className="px-4 py-2 text-right font-medium">{j.credit}</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y">
                        {e.lines.map((l) => (
                          <tr key={l.id}>
                            <td className="px-4 py-2 whitespace-nowrap">
                              <span className="font-mono text-xs">{l.accountCode}</span> {l.accountName}
                            </td>
                            <td className="px-4 py-2 text-muted-foreground">{l.memo || "–"}</td>
                            <td className="px-4 py-2 text-right">{l.entrySide === "DEBIT" ? <Money value={l.amount} /> : null}</td>
                            <td className="px-4 py-2 text-right">{l.entrySide === "DEBIT" ? null : <Money value={l.amount} />}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : null}
              </div>
            );
          })}
        </Card>
      )}
      {page.total > 0 ? <UrlPagination {...page} /> : null}
    </div>
  );
}
