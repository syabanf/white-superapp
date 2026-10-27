"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { CheckCheck, Pencil, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useErpAction } from "@/features/erp/shared/client";
import { ConfirmAction } from "@/features/erp/shared/confirm-action";
import { Money } from "@/features/erp/shared/money-display";
import { formatDate, formatDateTime } from "@/lib/format";
import { deleteJournalEntryAction, postJournalEntryAction } from "../actions";
import type { JournalEntry } from "../types";
import type { FinancePerms } from "../perms";
import { fs } from "../strings";

const j = fs.journals;

export function JournalActions({ entry, perms }: { entry: JournalEntry; perms: FinancePerms }) {
  const router = useRouter();
  const { run } = useErpAction();
  if (entry.status !== "DRAFT" || !entry.canEdit) return null;
  return (
    <>
      {perms.canWrite ? (
        <>
          <ConfirmAction
            trigger={
              <Button variant="ghost" className="text-destructive">
                <Trash2 className="size-4" /> {fs.delete}
              </Button>
            }
            title={fs.confirmDeleteTitle}
            description={j.deleteDesc}
            confirmLabel={fs.delete}
            destructive
            onConfirm={async () => {
              const res = await run(() => deleteJournalEntryAction(entry.id), { success: fs.deleted });
              if (res.ok) router.push("/erp/finance/journals");
              return res.ok;
            }}
          />
          <Button asChild variant="outline">
            <Link href={`/erp/finance/journals/${entry.id}/edit`}>
              <Pencil className="size-4" /> {fs.edit}
            </Link>
          </Button>
        </>
      ) : null}
      {perms.canApprove ? (
        <ConfirmAction
          trigger={
            <Button>
              <CheckCheck className="size-4" /> {j.post}
            </Button>
          }
          title={j.postTitle}
          description={j.postDesc}
          confirmLabel={j.post}
          onConfirm={async () => (await run(() => postJournalEntryAction(entry.id), { success: j.posted })).ok}
        />
      ) : null}
    </>
  );
}

function Meta({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <dt className="label-mono text-muted-foreground">{label}</dt>
      <dd className="mt-1 truncate text-sm">{children}</dd>
    </div>
  );
}

export function JournalDetail({ entry }: { entry: JournalEntry }) {
  return (
    <div className="page-stack">
      <Card className="px-(--card-pad)">
        <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Meta label={fs.date}>{formatDate(entry.entryDate)}</Meta>
          <Meta label={j.period}>
            {entry.fiscalPeriodName} {entry.fiscalYearCode ? `(${entry.fiscalYearCode})` : ""}
          </Meta>
          <Meta label={fs.type}>
            {fs.entryType[entry.entryType] ?? entry.entryType}
            {entry.isRecon ? ` · ${j.recon}` : ""}
          </Meta>
          <Meta label={j.source}>
            {entry.sourceModule ? `${entry.sourceModule} · ${entry.sourceEventCode}` : "–"}
          </Meta>
          {entry.postedAt ? (
            <Meta label={j.postedAt}>
              {formatDateTime(entry.postedAt)}
              {entry.postedBy ? ` · ${entry.postedBy}` : ""}
            </Meta>
          ) : null}
          <div className="sm:col-span-2 lg:col-span-4">
            <dt className="label-mono text-muted-foreground">{fs.description}</dt>
            <dd className="mt-1 text-sm whitespace-pre-wrap">{entry.description || fs.noDescription}</dd>
          </div>
        </dl>
      </Card>
      <Card className="gap-0 overflow-hidden p-0">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[640px] text-sm">
            <thead className="bg-muted/40 text-xs text-muted-foreground">
              <tr>
                <th className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2 text-left font-medium">{fs.account}</th>
                <th className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2 text-left font-medium">{fs.memo}</th>
                <th className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2 text-right font-medium">{fs.debit}</th>
                <th className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2 text-right font-medium">{fs.credit}</th>
              </tr>
            </thead>
            <tbody>
              {entry.lines.map((l) => (
                <tr key={l.id} className="border-t">
                  <td className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2.5">
                    <span className="mr-2 font-mono text-xs text-muted-foreground">{l.accountCode}</span>
                    {l.accountName}
                  </td>
                  <td className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2.5 text-muted-foreground">{l.memo || "–"}</td>
                  <td className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2.5 text-right">{l.entrySide === "DEBIT" ? <Money value={l.amount} /> : null}</td>
                  <td className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2.5 text-right">{l.entrySide === "CREDIT" ? <Money value={l.amount} /> : null}</td>
                </tr>
              ))}
            </tbody>
            <tfoot className="border-t bg-muted/30 font-semibold">
              <tr>
                <td colSpan={2} className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2.5 text-right text-xs text-muted-foreground">
                  {fs.total}
                </td>
                <td className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2.5 text-right">
                  <Money value={entry.totalDebit} />
                </td>
                <td className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2.5 text-right">
                  <Money value={entry.totalCredit} />
                </td>
              </tr>
            </tfoot>
          </table>
        </div>
      </Card>
    </div>
  );
}
