"use client";

import * as React from "react";
import Link from "next/link";
import { AlarmClock, MoreHorizontal, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useErpAction } from "@/features/erp/shared/client";
import { formatRupiah, parseMoney } from "@/features/erp/shared/money";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { moveDealAction } from "../actions";
import { daysSince, nameOf, type SalesCaps } from "../lib";
import { SERVICE_LINE, lbl, ss } from "../strings";
import type { Deal, LostReason, Stage } from "../types";
import { DealFormDialog, type DealFormRefs } from "./deal-form";
import { Fld, FormDialog, FormFooter, SimpleSelect } from "./ui";

const s = ss.pipeline;

type Pending = { deal: Deal; stage: Stage };

/** Kanban of deals by stage. Drag a card or use its menu to move it. */
export function PipelineBoard({
  stages,
  deals,
  lostReasons,
  refs,
  caps,
}: {
  stages: Stage[];
  deals: Deal[];
  lostReasons: LostReason[];
  refs: DealFormRefs;
  caps: SalesCaps;
}) {
  const names = Object.fromEntries(refs.owners.map((o) => [o.value, o.label]));
  const [, startTransition] = React.useTransition();
  const [board, moveLocal] = React.useOptimistic(deals, (state: Deal[], m: { id: string; stage: Stage }) =>
    state.map((d) => (d.id === m.id ? { ...d, stageId: m.stage.id, stageName: m.stage.name } : d)),
  );
  const [dragId, setDragId] = React.useState<string | null>(null);
  const [overStage, setOverStage] = React.useState<string | null>(null);
  const [closing, setClosing] = React.useState<Pending | null>(null);

  const move = (deal: Deal, stage: Stage) => {
    if (!caps.write || deal.stageId === stage.id) return;
    if (stage.isWon || stage.isLost) return setClosing({ deal, stage });
    startTransition(async () => {
      moveLocal({ id: deal.id, stage });
      const res = await moveDealAction(deal.id, stage.id);
      if (res.ok) toast.success(s.moved);
      else toast.error(res.error);
    });
  };

  const openDeals = board.filter((d) => !d.isWon && !d.isLost);
  const total = openDeals.reduce((acc, d) => acc + parseMoney(d.valueFinal || d.valueEstimate), 0);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-2 text-sm text-muted-foreground">
        <span>
          {s.openDeals(openDeals.length)} · {s.pipelineValue}{" "}
          <span className="font-semibold text-foreground tabular">{formatRupiah(total)}</span>
        </span>
        {caps.write ? (
          <DealFormDialog
            deal={null}
            refs={refs}
            trigger={
              <Button>
                <Plus className="size-4" /> {s.add}
              </Button>
            }
          />
        ) : null}
      </div>
      <div className="flex snap-x gap-3 overflow-x-auto pb-3 scrollbar-thin" role="list" aria-label={s.board}>
        {stages.map((stage) => {
          const items = board.filter((d) => d.stageId === stage.id);
          const value = items.reduce((acc, d) => acc + parseMoney(d.valueFinal || d.valueEstimate), 0);
          return (
            <section
              key={stage.id}
              role="listitem"
              aria-label={stage.name}
              onDragOver={(e) => {
                if (!dragId) return;
                e.preventDefault();
                setOverStage(stage.id);
              }}
              onDragLeave={() => setOverStage((cur) => (cur === stage.id ? null : cur))}
              onDrop={(e) => {
                e.preventDefault();
                const deal = board.find((d) => d.id === dragId);
                setDragId(null);
                setOverStage(null);
                if (deal) move(deal, stage);
              }}
              className={cn(
                "flex w-72 shrink-0 snap-start flex-col rounded-xl border bg-muted/30 transition-colors",
                overStage === stage.id && "border-brand bg-brand/5",
              )}
            >
              <header className="flex items-start justify-between gap-2 border-b px-3 py-2.5">
                <div className="min-w-0">
                  <h3
                    className={cn(
                      "truncate text-sm font-semibold",
                      stage.isWon && "text-positive",
                      stage.isLost && "text-destructive",
                    )}
                  >
                    {stage.name}
                  </h3>
                  <p className="text-xs text-muted-foreground tabular">
                    {items.length} · {formatRupiah(value)}
                  </p>
                </div>
                <span className="text-xs text-muted-foreground tabular">{stage.probability}%</span>
              </header>
              <div className="flex min-h-24 flex-col gap-2 p-2">
                {items.length === 0 ? (
                  <p className="px-1 py-6 text-center text-xs text-muted-foreground">
                    {dragId ? s.dropHere : s.emptyStage}
                  </p>
                ) : null}
                {items.map((deal) => {
                  const days = daysSince(deal.enteredStageAt);
                  const stuck = !deal.isWon && !deal.isLost && stage.stuckThresholdDays > 0 && days > stage.stuckThresholdDays;
                  return (
                    <article
                      key={deal.id}
                      draggable={caps.write}
                      onDragStart={(e) => {
                        setDragId(deal.id);
                        e.dataTransfer.effectAllowed = "move";
                        e.dataTransfer.setData("text/plain", deal.id);
                      }}
                      onDragEnd={() => {
                        setDragId(null);
                        setOverStage(null);
                      }}
                      className={cn(
                        "group rounded-lg border bg-card p-3 shadow-xs transition-shadow hover:shadow-[var(--lift-shadow)]",
                        caps.write && "cursor-grab active:cursor-grabbing",
                        dragId === deal.id && "opacity-50",
                      )}
                    >
                      <div className="flex items-start justify-between gap-2">
                        <Link
                          href={`/erp/sales/pipeline/${deal.id}`}
                          className="min-w-0 text-sm leading-snug font-medium hover:underline"
                        >
                          {deal.title}
                        </Link>
                        {caps.write ? (
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button size="icon-xs" variant="ghost" aria-label={s.stage} className="-mt-0.5 -mr-1 shrink-0">
                                <MoreHorizontal className="size-3.5" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuLabel>{s.stage}</DropdownMenuLabel>
                              {stages
                                .filter((st) => st.id !== deal.stageId)
                                .map((st) => (
                                  <DropdownMenuItem key={st.id} onSelect={() => move(deal, st)}>
                                    {st.name}
                                  </DropdownMenuItem>
                                ))}
                            </DropdownMenuContent>
                          </DropdownMenu>
                        ) : null}
                      </div>
                      <p className="mt-0.5 truncate text-xs text-muted-foreground">
                        {deal.clientName || deal.picName || ss.dash}
                      </p>
                      <div className="mt-2 flex items-center justify-between gap-2">
                        <span className="text-sm font-semibold tabular">
                          {formatRupiah(deal.valueFinal || deal.valueEstimate)}
                        </span>
                        <span className="truncate text-xs text-muted-foreground">{nameOf(names, deal.ownerUserId)}</span>
                      </div>
                      {deal.serviceLines.length || stuck ? (
                        <div className="mt-2 flex flex-wrap gap-1">
                          {stuck ? (
                            <span className="inline-flex items-center gap-1 rounded-full bg-amber-500/15 px-2 py-0.5 text-[11px] text-amber-700 dark:text-amber-400">
                              <AlarmClock className="size-3" /> {s.stuck(days)}
                            </span>
                          ) : null}
                          {deal.serviceLines.map((l) => (
                            <span key={l} className="rounded-full bg-muted px-2 py-0.5 text-[11px] text-muted-foreground">
                              {lbl(SERVICE_LINE, l)}
                            </span>
                          ))}
                        </div>
                      ) : null}
                    </article>
                  );
                })}
              </div>
            </section>
          );
        })}
      </div>
      <CloseDealDialog pending={closing} lostReasons={lostReasons} onClose={() => setClosing(null)} />
    </div>
  );
}

/** Won needs the final value; lost needs a reason. Also used from the deal page. */
export function CloseDealDialog({
  pending,
  lostReasons,
  onClose,
}: {
  pending: Pending | null;
  lostReasons: LostReason[];
  onClose: () => void;
}) {
  return (
    <FormDialog
      open={!!pending}
      onOpenChange={(o) => !o && onClose()}
      title={pending ? s.moveTitle(pending.stage.name) : ""}
      description={pending?.stage.isWon ? s.wonHint : undefined}
    >
      {(close) => (pending ? <CloseDealForm pending={pending} lostReasons={lostReasons} close={close} /> : null)}
    </FormDialog>
  );
}

function CloseDealForm({ pending, lostReasons, close }: { pending: Pending; lostReasons: LostReason[]; close: () => void }) {
  const { deal, stage } = pending;
  const [valueFinal, setValueFinal] = React.useState(deal.valueFinal || deal.valueEstimate || "");
  const [reason, setReason] = React.useState(deal.lostReasonId || "");
  const [error, setError] = React.useState<string[]>();
  const { pending: busy, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        if (stage.isWon && parseMoney(valueFinal) <= 0) return setError(["Wajib diisi"]);
        if (stage.isLost && !reason) return setError(["Wajib diisi"]);
        void run(
          () => moveDealAction(deal.id, stage.id, stage.isWon ? { valueFinal } : { lostReasonId: reason }),
          { success: s.moved, onOk: close },
        );
      }}
    >
      <p className="text-sm font-medium">{deal.title}</p>
      {stage.isWon ? (
        <Fld label={`${s.valueFinal} *`} htmlFor="close-value" error={error}>
          <MoneyInput id="close-value" value={valueFinal} onChange={setValueFinal} invalid={!!error} />
        </Fld>
      ) : (
        <Fld label={`${s.lostReason} *`} htmlFor="close-reason" error={error}>
          <SimpleSelect
            id="close-reason"
            value={reason}
            onChange={setReason}
            options={lostReasons.map((r) => ({ value: r.id, label: r.name }))}
            invalid={!!error}
          />
        </Fld>
      )}
      <FormFooter onCancel={close} pending={busy} submitLabel={ss.confirm} />
    </form>
  );
}
