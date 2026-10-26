"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2, Play } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import type { ColMeta } from "@/components/dashboard/data-table";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { useErpAction } from "@/features/erp/shared/client";
import { formatDecimal, formatNumber } from "@/lib/format";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { runSnapshotAction, saveManualAction, setScorecardStatusAction } from "../actions";
import { scoreTone } from "../lib";
import { sp } from "../strings";
import type { Indicator, ListScorecardsResponse, Recommendation, Scorecard, ScorecardDetail } from "../types";
import { ConfirmAction, DRAFT_FINAL_KINDS, DRAFT_FINAL_LABELS, FormRow, Score } from "./common";
import { useResource } from "./use-resource";

const k = sp.kpi;

export function RunSnapshotButton({ year, month }: { year: number; month: number }) {
  const { pending, run } = useErpAction();
  return (
    <ConfirmAction
      trigger={
        <Button size="sm" disabled={pending}>
          {pending ? <Loader2 className="size-4 animate-spin" /> : <Play className="size-4" />}
          {pending ? k.running : k.runSnapshot}
        </Button>
      }
      title={k.runTitle(sp.monthLabel(year, month))}
      description={k.runBody}
      confirmLabel={k.run}
      onConfirm={() => run(() => runSnapshotAction(year, month), { onOk: (r) => toast.success(k.snapshotDone(r)) })}
    />
  );
}

/** Scorecards of a month with the 3-month average used for promotion recommendations. */
export function ScorecardsTable({
  data,
  recommendations,
  canApprove,
}: {
  data: ListScorecardsResponse;
  recommendations: Recommendation[];
  canApprove: boolean;
}) {
  const [openId, setOpenId] = React.useState<string | null>(null);
  const recBy = React.useMemo(() => new Map(recommendations.map((r) => [r.employeeId, r])), [recommendations]);
  const columns: ColumnDef<Scorecard, unknown>[] = [
    {
      id: "employee",
      accessorFn: (r) => `${r.employeeName} ${r.departmentName}`,
      header: sp.common.employee,
      cell: ({ row }) => (
        <span className="min-w-0">
          <span className="block truncate font-medium">{row.original.employeeName}</span>
          <span className="block truncate text-xs text-muted-foreground">{row.original.departmentName}</span>
        </span>
      ),
    },
    {
      id: "score",
      accessorFn: (r) => r.score ?? -1,
      header: sp.common.score,
      meta: { align: "right" } satisfies ColMeta,
      cell: ({ row }) =>
        row.original.score == null ? (
          <span className="text-xs text-muted-foreground">{sp.common.noScore}</span>
        ) : (
          <Score value={row.original.score} digits={2} />
        ),
    },
    {
      id: "avg",
      accessorFn: (r) => recBy.get(r.employeeId)?.avgScore ?? -1,
      header: k.avg3,
      meta: { align: "right" } satisfies ColMeta,
      cell: ({ row }) => {
        const r = recBy.get(row.original.employeeId);
        return r ? (
          <span className="tabular" title={k.avgHint(r.periods)}>
            {formatDecimal(r.avgScore, 1)}
          </span>
        ) : (
          <span className="text-muted-foreground">–</span>
        );
      },
    },
    {
      accessorKey: "status",
      header: sp.common.status,
      cell: ({ row }) => <ErpStatusBadge status={row.original.status} labels={DRAFT_FINAL_LABELS} kinds={DRAFT_FINAL_KINDS} />,
    },
  ];
  return (
    <>
      <ErpDataTable
        columns={columns}
        data={data.scorecards}
        emptyMessage={k.empty}
        exportName="scorecards"
        getRowId={(r) => r.id}
        onRowClick={(row) => setOpenId(row.original.id)}
      />
      <ScorecardDialog id={openId} indicators={data.indicators} canApprove={canApprove} onClose={() => setOpenId(null)} />
    </>
  );
}

const pctOf = (fraction?: number) => (fraction == null ? "–" : `${formatNumber(fraction * 100, 1)}%`);

/** Breakdown, 12-month history, supervisor rubric, manual actuals and finalize/reopen. */
function ScorecardDialog({
  id,
  indicators,
  canApprove,
  onClose,
}: {
  id: string | null;
  indicators: Indicator[];
  canApprove: boolean;
  onClose: () => void;
}) {
  const detail = useResource<ScorecardDetail>(id ? `/v1/performance/kpi/scorecards/${encodeURIComponent(id)}` : null);
  const employeeId = detail.data?.scorecard.employeeId;
  const history = useResource<ListScorecardsResponse>(
    employeeId ? `/v1/performance/kpi/scorecards?employeeId=${encodeURIComponent(employeeId)}&history=12` : null,
  );
  const current = detail.data;
  const error = detail.error;
  const reload = async () => {
    await Promise.all([detail.reload(), history.reload()]);
  };

  return (
    <Dialog open={!!id} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-h-[88vh] overflow-y-auto sm:max-w-3xl">
        {error ? (
          <DialogHeader>
            <DialogTitle>{sp.common.loadFailed}</DialogTitle>
            <DialogDescription>{error}</DialogDescription>
          </DialogHeader>
        ) : !current ? (
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Loader2 className="size-4 animate-spin" /> {sp.common.loading}
            </DialogTitle>
          </DialogHeader>
        ) : (
          <ScorecardBody key={current.scorecard.id + current.scorecard.status} detail={current} history={history.data?.scorecards ?? []} indicators={indicators} canApprove={canApprove} reload={reload} />
        )}
      </DialogContent>
    </Dialog>
  );
}

function ScorecardBody({
  detail,
  history,
  indicators,
  canApprove,
  reload,
}: {
  detail: ScorecardDetail;
  history: Scorecard[];
  indicators: Indicator[];
  canApprove: boolean;
  reload: () => Promise<void>;
}) {
  const sc = detail.scorecard;
  const { pending, run } = useErpAction();
  const nameByCode = new Map([
    ...indicators.map((i) => [i.code, i.name] as const),
    ...detail.snapshots.map((s) => [s.indicatorCode, s.indicatorName] as const),
  ]);
  const snapByCode = new Map(detail.snapshots.map((s) => [s.indicatorCode, s]));
  const excluded = sc.breakdown.filter((r) => r.attainment == null);
  const rubric = sc.breakdown.find((r) => r.code === "supervisor_rubric");
  const manual = detail.snapshots.filter((s) => s.sourceKind === "manual" && s.indicatorCode !== "supervisor_rubric");
  const editable = sc.status === "draft" && detail.canRate;
  const [rubricValue, setRubricValue] = React.useState(0);
  const [rubricNotes, setRubricNotes] = React.useState("");
  const [actuals, setActuals] = React.useState<Record<string, string>>({});
  const period = { periodYear: sc.periodYear, periodMonth: sc.periodMonth };

  const manualDone = (r: { status: string; message: string }) => {
    toast.success(k.manualStatus[r.status] ?? r.message);
    void reload();
  };

  return (
    <div className="flex flex-col gap-5">
      <DialogHeader>
        <DialogTitle className="flex flex-wrap items-center gap-2">
          {k.detailTitle(sc.employeeName, sp.monthLabel(sc.periodYear, sc.periodMonth))}
          <ErpStatusBadge status={sc.status} labels={DRAFT_FINAL_LABELS} kinds={DRAFT_FINAL_KINDS} />
        </DialogTitle>
        <DialogDescription>{sc.departmentName}</DialogDescription>
      </DialogHeader>

      <div className="flex items-baseline gap-3">
        <span className={cn("text-4xl font-bold tracking-tight tabular", scoreTone(sc.score))}>
          {sc.score == null ? "–" : formatDecimal(sc.score, 2)}
        </span>
        <span className="text-sm text-muted-foreground">/ 100</span>
        {sc.rawScore != null && sc.rawScore > 100 ? (
          <span className="text-xs text-muted-foreground">{k.overAchieve(formatDecimal(sc.rawScore, 2))}</span>
        ) : null}
      </div>

      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{k.indicator}</TableHead>
              <TableHead className="text-right">{k.actual}</TableHead>
              <TableHead className="hidden text-right sm:table-cell">{k.target}</TableHead>
              <TableHead className="text-right">{k.weight}</TableHead>
              <TableHead className="text-right">{k.attainment}</TableHead>
              <TableHead className="text-right">{k.contribution}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sc.breakdown.map((r) => {
              const snap = snapByCode.get(r.code);
              return (
                <TableRow key={r.code} className={r.attainment == null ? "opacity-50" : undefined}>
                  <TableCell className="whitespace-normal">{nameByCode.get(r.code) ?? r.code}</TableCell>
                  <TableCell className="text-right tabular">{snap?.actual == null ? "–" : formatNumber(snap.actual, 2)}</TableCell>
                  <TableCell className="hidden text-right tabular sm:table-cell">{snap?.target == null ? "–" : formatNumber(snap.target, 2)}</TableCell>
                  <TableCell className="text-right tabular" title={`${k.effectiveWeight}: ${formatNumber(r.effectiveWeight, 2)}`}>
                    {formatNumber(r.weight)}
                  </TableCell>
                  <TableCell className="text-right tabular">{pctOf(r.attainment)}</TableCell>
                  <TableCell className="text-right tabular">{r.attainment == null ? "–" : formatNumber(r.contribution, 2)}</TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>
      {excluded.length > 0 ? (
        <p className="text-xs text-muted-foreground">
          {k.excluded(excluded.length, excluded.map((r) => nameByCode.get(r.code) ?? r.code).join(", "))}
        </p>
      ) : null}

      {history.length > 1 ? (
        <section className="flex flex-col gap-2">
          <h3 className="text-sm font-semibold">{k.history}</h3>
          <div className="flex gap-1.5 overflow-x-auto pb-1">
            {[...history]
              .sort((a, b) => a.periodYear * 12 + a.periodMonth - (b.periodYear * 12 + b.periodMonth))
              .map((h) => (
                <span key={h.id} className="min-w-14 rounded-lg bg-muted/60 px-2 py-1 text-center text-xs">
                  <span className="block text-[10px] text-muted-foreground">
                    {sp.monthShort(h.periodMonth)} {String(h.periodYear).slice(2)}
                  </span>
                  <Score value={h.score} digits={0} />
                </span>
              ))}
          </div>
        </section>
      ) : null}

      {editable ? (
        <section className="flex flex-col gap-2 border-t pt-4">
          <h3 className="text-sm font-semibold">
            {k.rubric}
            {rubric?.attainment != null ? (
              <span className="ml-2 text-xs font-normal text-muted-foreground">{k.rubricSaved(formatDecimal(rubric.attainment * 5, 1))}</span>
            ) : null}
          </h3>
          <div className="flex gap-2">
            {[1, 2, 3, 4, 5].map((v) => (
              <Button
                key={v}
                type="button"
                variant={rubricValue === v ? "default" : "outline"}
                className="h-11 flex-1"
                aria-pressed={rubricValue === v}
                onClick={() => setRubricValue(v)}
              >
                {v}
              </Button>
            ))}
          </div>
          <Textarea value={rubricNotes} onChange={(e) => setRubricNotes(e.target.value)} placeholder={k.rubricNotes} aria-label={k.rubricNotes} />
          <Button
            size="sm"
            className="self-start"
            disabled={pending || rubricValue === 0}
            onClick={() =>
              run(() => saveManualAction({ employeeId: sc.employeeId, ...period, value: rubricValue, notes: rubricNotes || undefined }), {
                onOk: manualDone,
              })
            }
          >
            {k.saveRubric}
          </Button>
        </section>
      ) : null}

      {editable && manual.length > 0 ? (
        <section className="flex flex-col gap-2 border-t pt-4">
          <h3 className="text-sm font-semibold">{k.manualActuals}</h3>
          <p className="text-xs text-muted-foreground">{k.manualHint}</p>
          <div className="grid gap-3 sm:grid-cols-2">
            {manual.map((m) => (
              <FormRow key={m.indicatorId} label={m.indicatorName} htmlFor={`act-${m.indicatorId}`}>
                <div className="flex gap-2">
                  <Input
                    id={`act-${m.indicatorId}`}
                    type="number"
                    step="any"
                    inputMode="decimal"
                    placeholder={m.actual == null ? "" : String(m.actual)}
                    value={actuals[m.indicatorId] ?? ""}
                    onChange={(e) => setActuals((a) => ({ ...a, [m.indicatorId]: e.target.value }))}
                  />
                  <Button
                    size="sm"
                    variant="secondary"
                    disabled={pending || !actuals[m.indicatorId]}
                    onClick={() =>
                      run(
                        () =>
                          saveManualAction({
                            employeeId: sc.employeeId,
                            indicatorId: m.indicatorId,
                            ...period,
                            value: Number(actuals[m.indicatorId]),
                          }),
                        { onOk: manualDone },
                      )
                    }
                  >
                    {sp.common.save}
                  </Button>
                </div>
              </FormRow>
            ))}
          </div>
        </section>
      ) : null}

      {canApprove ? (
        <div className="flex justify-end border-t pt-4">
          <Button
            variant={sc.status === "draft" ? "default" : "outline"}
            disabled={pending}
            onClick={() =>
              run(() => setScorecardStatusAction(sc.id, sc.status === "draft" ? "finalize" : "reopen"), {
                success: sc.status === "draft" ? k.finalized : k.reopened,
                onOk: () => void reload(),
              })
            }
          >
            {sc.status === "draft" ? k.finalize : k.reopen}
          </Button>
        </div>
      ) : null}
    </div>
  );
}
