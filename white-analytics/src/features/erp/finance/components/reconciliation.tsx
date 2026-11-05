"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import type { ColumnDef } from "@tanstack/react-table";
import { Ban, CheckCheck, Loader2, RefreshCw, RotateCcw, Upload } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ActionButton, ConfirmAction, Fld, FormDialog, FormFooter, ReasonDialog } from "@/features/erp/sales/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable, type ServerPage } from "@/features/erp/shared/data-table";
import { FileUpload } from "@/features/erp/shared/file-upload";
import { Money } from "@/features/erp/shared/money-display";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { DateRangeFilter } from "@/features/erp/shared/period-picker";
import { ResetFilters, SelectFilter } from "@/features/erp/shared/url-filters";
import { formatDate } from "@/lib/format";
import { ignoreLineAction, matchCandidatesAction, matchLineAction, recordLineAction, rematchAction, reopenLineAction } from "../actions";
import { fs } from "../strings";
import type { MatchCandidate, PostableAccount, StatementImportResult, StatementLine } from "../types";
import { AccountPicker } from "./account-picker";

const s = fs.recon;

export function StatementUpload({ accountId }: { accountId: string }) {
  const router = useRouter();
  const [result, setResult] = React.useState<StatementImportResult>();
  return (
    <FormDialog
      title={s.upload}
      description={s.uploadHint}
      trigger={
        <Button size="sm">
          <Upload className="size-4" /> {s.upload}
        </Button>
      }
    >
      {() => (
        <div className="flex flex-col gap-3">
          <FileUpload<StatementImportResult>
            url={`/api/erp/api/finance/recon/accounts/${encodeURIComponent(accountId)}/statements`}
            accept=".csv,text/csv"
            acceptLabel="CSV"
            maxMb={5}
            multiple={false}
            onUploaded={(r) => {
              setResult(r);
              router.refresh();
            }}
          />
          {result ? (
            <p role="status" className="text-sm">
              {result.inserted} baris baru, {result.skipped} dilewati. {s.rematched(result.matched, result.suggested, result.unmatched)}.
            </p>
          ) : null}
        </div>
      )}
    </FormDialog>
  );
}

export function RematchButton({ accountId }: { accountId: string }) {
  return (
    <ActionButton size="sm" variant="outline" action={() => rematchAction(accountId)} success={s.rematch}>
      <RefreshCw className="size-4" /> {s.rematch}
    </ActionButton>
  );
}

function VerifyPanel({ line, accounts, canApprove, close }: { line: StatementLine; accounts: PostableAccount[]; canApprove: boolean; close: () => void }) {
  const [candidates, setCandidates] = React.useState<MatchCandidate[]>();
  const [counter, setCounter] = React.useState("");
  const [description, setDescription] = React.useState(line.description);
  const { pending, run } = useErpAction();
  React.useEffect(() => {
    void matchCandidatesAction(line.id).then((r) => setCandidates(r.ok ? (r.data.candidates ?? []) : []));
  }, [line.id]);
  return (
    <div className="flex flex-col gap-5">
      <section className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold">{s.candidates}</h3>
        {!candidates ? (
          <Loader2 className="size-4 animate-spin text-muted-foreground" />
        ) : candidates.length === 0 ? (
          <p className="text-sm text-muted-foreground">{s.noCandidates}</p>
        ) : (
          <ul className="divide-y rounded-lg border text-sm">
            {candidates.map((c) => {
              const needsApproval = c.kind === "INVOICE" && !canApprove;
              return (
                <li key={c.kind + c.ref} className="flex items-center gap-3 px-3 py-2">
                  <span className="min-w-0 flex-1">
                    <span className="block truncate font-medium">{c.label}</span>
                    <span className="block text-xs text-muted-foreground">
                      {c.kind === "INVOICE" ? s.invoice : s.journal} · {c.date ? `${formatDate(c.date)} · ` : ""}
                      {c.reason} · skor {c.score}
                    </span>
                    {c.kind === "INVOICE" ? <span className="block text-xs text-muted-foreground">{s.invoiceHint}</span> : null}
                  </span>
                  <Button
                    size="xs"
                    disabled={pending || needsApproval}
                    onClick={() => void run(() => matchLineAction(line.id, c.kind, c.ref), { success: s.matched, onOk: close })}
                  >
                    {s.use}
                  </Button>
                </li>
              );
            })}
          </ul>
        )}
      </section>
      {canApprove ? (
        <form
          className="flex flex-col gap-3 border-t pt-4"
          onSubmit={(e) => {
            e.preventDefault();
            void run(() => recordLineAction(line.id, counter, description), { success: s.recorded, onOk: close });
          }}
        >
          <div>
            <h3 className="text-sm font-semibold">{s.record}</h3>
            <p className="text-xs text-muted-foreground">{s.recordHint}</p>
          </div>
          <Fld label={s.counter} htmlFor="rec-counter">
            <AccountPicker accounts={accounts} value={counter} onChange={setCounter} />
          </Fld>
          <Fld label={s.description} htmlFor="rec-desc">
            <Input id="rec-desc" value={description} onChange={(e) => setDescription(e.target.value)} />
          </Fld>
          <FormFooter onCancel={close} pending={pending} submitLabel={s.record} />
        </form>
      ) : null}
    </div>
  );
}

export function StatementLines({
  lines,
  page,
  accounts,
  canWrite,
  canApprove,
}: {
  lines: StatementLine[];
  page: ServerPage;
  accounts: PostableAccount[];
  canWrite: boolean;
  canApprove: boolean;
}) {
  const columns: ColumnDef<StatementLine, unknown>[] = [
    { accessorKey: "lineDate", header: s.date, cell: ({ row }) => formatDate(row.original.lineDate) },
    {
      accessorKey: "description",
      header: s.description,
      cell: ({ row }) => (
        <div className="max-w-md min-w-0">
          <div className="line-clamp-2">{row.original.description || "–"}</div>
          {row.original.senderName ? <div className="text-xs text-muted-foreground">{row.original.senderName}</div> : null}
        </div>
      ),
    },
    { accessorKey: "amount", header: s.amount, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.amount} signed /> },
    {
      accessorKey: "status",
      header: s.status,
      cell: ({ row }) => <ErpStatusBadge status={row.original.status} labels={s.statuses} />,
    },
    {
      id: "result",
      header: s.result,
      enableSorting: false,
      cell: ({ row }) => {
        const l = row.original;
        if (l.status === "MATCHED") {
          return (
            <span className="text-xs">
              {l.matchedEntryId ? (
                <Link href={`/erp/finance/journals/${l.matchedEntryId}`} className="font-mono hover:underline">
                  {l.matchedEntryNo || s.journal}
                </Link>
              ) : (
                l.note
              )}
              {l.verifiedBy ? null : <span className="text-muted-foreground"> · {s.auto}</span>}
            </span>
          );
        }
        if (l.status === "IGNORED") return <span className="text-xs text-muted-foreground">{l.note}</span>;
        if (l.status === "SUGGESTED") {
          return (
            <span className="block max-w-xs text-xs">
              {l.suggestedLabel}
              <span className="block text-muted-foreground">{l.suggestedReason}</span>
            </span>
          );
        }
        return null;
      },
    },
    {
      id: "actions",
      header: "",
      enableSorting: false,
      cell: ({ row }) => {
        const l = row.original;
        if (!canWrite) return null;
        if (l.status === "MATCHED" || l.status === "IGNORED") {
          return (
            <div className="flex justify-end">
              <ConfirmAction
                title={s.reopenTitle}
                description={s.reopenBody}
                success={s.reopened}
                destructive={false}
                action={() => reopenLineAction(l.id)}
                trigger={
                  <Button size="xs" variant="ghost">
                    <RotateCcw className="size-3.5" /> {s.reopen}
                  </Button>
                }
              />
            </div>
          );
        }
        return (
          <div className="flex justify-end gap-1">
            <FormDialog
              title={s.verifyTitle}
              description={`${formatDate(l.lineDate)} · ${l.description}`}
              trigger={
                <Button size="xs">
                  <CheckCheck className="size-3.5" /> {s.verify}
                </Button>
              }
            >
              {(close) => <VerifyPanel line={l} accounts={accounts} canApprove={canApprove} close={close} />}
            </FormDialog>
            <ReasonDialog
              title={s.ignore}
              label={s.ignoreLabel}
              submitLabel={s.ignore}
              success={s.ignored}
              onSubmit={(note) => ignoreLineAction(l.id, note)}
              trigger={
                <Button size="xs" variant="ghost">
                  <Ban className="size-3.5" /> {s.ignore}
                </Button>
              }
            />
          </div>
        );
      },
    },
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={lines}
      pagination={page}
      getRowId={(l) => l.id}
      emptyMessage={s.empty}
      toolbar={
        <>
          <SelectFilter
            param="status"
            label={s.status}
            allLabel={s.allStatus}
            options={Object.entries(s.statuses).map(([value, label]) => ({ value, label }))}
          />
          <DateRangeFilter />
          <ResetFilters params={["status", "from", "to"]} />
        </>
      }
    />
  );
}
