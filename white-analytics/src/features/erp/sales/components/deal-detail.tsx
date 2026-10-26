"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { ArrowRightLeft, Loader2, Pencil, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { useErpAction } from "@/features/erp/shared/client";
import { cn } from "@/lib/utils";
import { deleteDealAction, moveDealAction, setDealMembersAction } from "../actions";
import { nameOf, sumsTo100 } from "../lib";
import { MEMBER_ROLE, lbl, opts, ss } from "../strings";
import type { Deal, DealMember, LostReason, Stage } from "../types";
import { DealFormDialog, type DealFormRefs } from "./deal-form";
import { CloseDealDialog } from "./pipeline-board";
import { ConfirmAction, type Option, Picker, SimpleSelect } from "./ui";

/** Edit, move and delete buttons on the deal page. */
export function DealHeaderActions({
  deal,
  stages,
  lostReasons,
  refs,
}: {
  deal: Deal;
  stages: Stage[];
  lostReasons: LostReason[];
  refs: DealFormRefs;
}) {
  const router = useRouter();
  const [closing, setClosing] = React.useState<{ deal: Deal; stage: Stage } | null>(null);
  const { pending, run } = useErpAction();
  return (
    <>
      <ConfirmAction
        title={ss.pipeline.deleteTitle}
        action={() => deleteDealAction(deal.id)}
        onDone={() => router.push("/erp/sales/pipeline")}
        trigger={
          <Button size="sm" variant="ghost">
            <Trash2 className="size-3.5" /> {ss.delete}
          </Button>
        }
      />
      <DealFormDialog
        deal={deal}
        refs={refs}
        trigger={
          <Button size="sm" variant="outline">
            <Pencil className="size-3.5" /> {ss.edit}
          </Button>
        }
      />
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button size="sm" disabled={pending}>
            {pending ? <Loader2 className="size-3.5 animate-spin" /> : <ArrowRightLeft className="size-3.5" />}
            {ss.pipeline.stage}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel>{deal.stageName}</DropdownMenuLabel>
          {stages
            .filter((st) => st.id !== deal.stageId)
            .map((st) => (
              <DropdownMenuItem
                key={st.id}
                onSelect={() =>
                  st.isWon || st.isLost
                    ? setClosing({ deal, stage: st })
                    : void run(() => moveDealAction(deal.id, st.id), { success: ss.pipeline.moved })
                }
              >
                {st.name}
              </DropdownMenuItem>
            ))}
        </DropdownMenuContent>
      </DropdownMenu>
      <CloseDealDialog pending={closing} lostReasons={lostReasons} onClose={() => setClosing(null)} />
    </>
  );
}

/** Deal team with roles and the revenue split (must total 100%). */
export function DealMembers({
  dealId,
  members,
  owners,
  canWrite,
}: {
  dealId: string;
  members: DealMember[];
  owners: Option[];
  canWrite: boolean;
}) {
  const d = ss.deal;
  const names = Object.fromEntries(owners.map((o) => [o.value, o.label]));
  const [rows, setRows] = React.useState(members);
  const { pending, run } = useErpAction();
  const total = Math.round(rows.reduce((a, r) => a + (Number(r.splitPercent) || 0), 0) * 100) / 100;
  const valid = rows.length === 0 || sumsTo100(rows.map((r) => r.splitPercent));
  const dirty = JSON.stringify(rows) !== JSON.stringify(members);

  if (!canWrite) {
    return rows.length ? (
      <ul className="space-y-1.5 text-sm">
        {rows.map((r) => (
          <li key={r.userId} className="flex justify-between gap-2">
            <span>
              {nameOf(names, r.userId)} <span className="text-xs text-muted-foreground">{lbl(MEMBER_ROLE, r.role)}</span>
            </span>
            <span className="tabular">{r.splitPercent}%</span>
          </li>
        ))}
      </ul>
    ) : (
      <p className="text-sm text-muted-foreground">{ss.none}</p>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {rows.map((r, i) => (
        <div key={i} className="grid grid-cols-[minmax(0,1fr)_72px_auto] items-center gap-2 sm:grid-cols-[minmax(0,1fr)_130px_72px_auto]">
          <div className="col-span-3 min-w-0 sm:col-span-1">
            <Picker
              value={r.userId}
              onChange={(v) => setRows((l) => l.map((x, j) => (j === i ? { ...x, userId: v } : x)))}
              options={owners}
              placeholder={d.member}
            />
          </div>
          <SimpleSelect
            value={r.role}
            onChange={(v) => setRows((l) => l.map((x, j) => (j === i ? { ...x, role: v } : x)))}
            options={opts(MEMBER_ROLE)}
          />
          <Input
            type="number"
            min={0}
            max={100}
            step="any"
            aria-label={d.split}
            value={r.splitPercent}
            onChange={(e) => setRows((l) => l.map((x, j) => (j === i ? { ...x, splitPercent: Number(e.target.value) } : x)))}
            className="tabular"
          />
          <Button size="icon-sm" variant="ghost" aria-label={ss.delete} onClick={() => setRows((l) => l.filter((_, j) => j !== i))}>
            <Trash2 className="size-3.5" />
          </Button>
        </div>
      ))}
      <div className="flex flex-wrap items-center justify-between gap-2 pt-1">
        <Button
          size="sm"
          variant="outline"
          onClick={() => setRows((l) => [...l, { userId: "", role: l.length ? "support" : "owner", splitPercent: l.length ? 0 : 100 }])}
        >
          <Plus className="size-3.5" /> {d.addMember}
        </Button>
        <span className={cn("text-xs tabular", valid ? "text-muted-foreground" : "text-destructive")}>
          {valid ? d.splitTotal(total) : d.splitInvalid}
        </span>
        <Button
          size="sm"
          disabled={!dirty || !valid || pending || rows.some((r) => !r.userId)}
          onClick={() => void run(() => setDealMembersAction(dealId, rows), { success: ss.saved })}
        >
          {pending ? <Loader2 className="size-3.5 animate-spin" /> : null}
          {d.saveMembers}
        </Button>
      </div>
    </div>
  );
}
