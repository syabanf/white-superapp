"use client";

import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Toggle } from "@/components/ui/toggle";
import {
  BOARD_FILTER_KEYS,
  BOARD_STAGES,
  chipParams,
  describeBoardFilters,
  type BoardFilters,
  type BoardStage,
} from "@/features/content/board-filters";
import { ITEM_STATUSES } from "@/features/content/lib";
import { c, cbf } from "@/features/content/strings";
import type { ItemStatus } from "@/features/content/types";
import { OptionSelect } from "../fields";
import { useSetParams } from "../use-set-param";

/**
 * Status, stage, date and overdue filters for the board, plus removable chips
 * for every active filter (including the ones only a dashboard drill sets:
 * platform, format, pillar, revision rounds, weekday and hour).
 */
export function BoardFilterBar({
  filters,
  shown,
  total,
}: {
  filters: BoardFilters;
  shown: number;
  total: number;
}) {
  const setParams = useSetParams();
  const chips = describeBoardFilters(filters);
  const singleStatus = filters.status.length === 1 ? filters.status[0]! : "";

  return (
    <>
      <OptionSelect<ItemStatus>
        label={cbf.status}
        value={singleStatus}
        onChange={(v) => setParams({ status: v })}
        allLabel={cbf.allStatuses}
        options={ITEM_STATUSES.map((s) => ({ value: s, label: c.itemStatus[s] }))}
        className="w-[150px]"
      />
      <OptionSelect<BoardStage>
        label={cbf.stage}
        value={filters.stage}
        onChange={(v) => setParams({ stage: v })}
        allLabel={cbf.allStages}
        options={BOARD_STAGES.map((s) => ({ value: s, label: cbf.stages[s] }))}
        className="w-[150px]"
      />
      <div className="flex items-center gap-1.5">
        <Input
          type="date"
          aria-label={cbf.dateFrom}
          value={filters.dateFrom}
          max={filters.dateTo || undefined}
          onChange={(e) => setParams({ dateFrom: e.target.value })}
          className="h-8 w-[9.5rem] rounded-full text-xs"
        />
        <span className="text-xs text-muted-foreground">–</span>
        <Input
          type="date"
          aria-label={cbf.dateTo}
          value={filters.dateTo}
          min={filters.dateFrom || undefined}
          onChange={(e) => setParams({ dateTo: e.target.value })}
          className="h-8 w-[9.5rem] rounded-full text-xs"
        />
      </div>
      <Toggle
        size="sm"
        variant="outline"
        pressed={filters.overdue}
        onPressedChange={(on) => setParams({ overdue: on ? "1" : null })}
        className="rounded-full data-[state=on]:border-negative/40 data-[state=on]:text-negative"
      >
        {cbf.overdue}
      </Toggle>

      {chips.length > 0 ? (
        <div className="flex w-full flex-wrap items-center gap-1.5" aria-label={cbf.active}>
          <span className="text-xs text-muted-foreground">{cbf.shown(shown, total)}</span>
          {chips.map((chip) => (
            <span
              key={chip.key}
              className="inline-flex max-w-full items-center gap-1 rounded-full bg-brand/10 py-0.5 pr-0.5 pl-2.5 text-xs"
            >
              <span className="truncate">{chip.label}</span>
              <button
                type="button"
                aria-label={cbf.remove(chip.label)}
                onClick={() => setParams(Object.fromEntries(chipParams(chip.key).map((k) => [k, null])))}
                className="flex size-5 shrink-0 items-center justify-center rounded-full outline-none hover:bg-brand/15 focus-visible:ring-2 focus-visible:ring-ring"
              >
                <X className="size-3" aria-hidden />
              </button>
            </span>
          ))}
          <Button
            variant="ghost"
            size="xs"
            onClick={() => setParams(Object.fromEntries(BOARD_FILTER_KEYS.map((k) => [k, null])))}
          >
            {cbf.clear}
          </Button>
        </div>
      ) : null}
    </>
  );
}
