"use client";

import * as React from "react";
import Link from "next/link";
import { FileText } from "lucide-react";
import { useErpAction } from "@/features/erp/shared/client";
import type { SearchOption } from "@/features/erp/shared/searchable-select";
import { ResetFilters, SearchFilter, SearchSelectFilter } from "@/features/erp/shared/url-filters";
import { formatDate } from "@/lib/format";
import { cn } from "@/lib/utils";
import { moveStageAction } from "../actions";
import { CANDIDATE_STATUS, SOURCE, rs } from "../strings";
import type { Candidate, PipelineColumn } from "../types";
import { StageMenu } from "./candidate-header";

/** Kanban by stage; writers drag cards between columns (or use the stage menu on touch screens). */
export function PipelineBoard({ columns: initial, jobs, canWrite }: { columns: PipelineColumn[]; jobs: SearchOption[]; canWrite: boolean }) {
  const [columns, setColumns] = React.useState(initial);
  const [over, setOver] = React.useState<string | null>(null);
  const { run } = useErpAction();
  // server refreshes replace the optimistic board
  const [synced, setSynced] = React.useState(initial);
  if (synced !== initial) {
    setSynced(initial);
    setColumns(initial);
  }

  const move = (candidateId: string, to: string) => {
    const from = columns.find((c) => c.candidates.some((x) => x.id === candidateId));
    if (!from || from.status === to) return;
    const card = from.candidates.find((x) => x.id === candidateId)!;
    const snapshot = columns;
    setColumns((cols) =>
      cols.map((c) =>
        c.status === from.status
          ? { ...c, count: c.count - 1, candidates: c.candidates.filter((x) => x.id !== candidateId) }
          : c.status === to
            ? { ...c, count: c.count + 1, candidates: [{ ...card, status: to }, ...c.candidates] }
            : c,
      ),
    );
    void run(() => moveStageAction(candidateId, to), { success: rs.candidates.moved, onFail: () => setColumns(snapshot) });
  };

  return (
    <div className="flex min-w-0 flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <SearchFilter placeholder={rs.pipeline.search} />
        <SearchSelectFilter param="job" options={jobs} placeholder={rs.candidates.allJobs} className="w-56" />
        <ResetFilters params={["q", "job"]} />
      </div>
      <div className="overflow-x-auto pb-2 scrollbar-thin">
        <div className="flex w-max gap-3">
          {columns.map((col) => (
            <section
              key={col.status}
              aria-label={CANDIDATE_STATUS[col.status] ?? col.label}
              onDragOver={(e) => {
                if (!canWrite) return;
                e.preventDefault();
                setOver(col.status);
              }}
              onDragLeave={() => setOver((o) => (o === col.status ? null : o))}
              onDrop={(e) => {
                e.preventDefault();
                setOver(null);
                const id = e.dataTransfer.getData("text/plain");
                if (id) move(id, col.status);
              }}
              className={cn(
                "flex w-64 shrink-0 flex-col gap-2 rounded-xl bg-muted/50 p-2 transition-colors",
                over === col.status && "bg-brand/10 ring-2 ring-brand/40",
              )}
            >
              <header className="flex items-center justify-between px-1.5 py-1 text-sm font-medium">
                {CANDIDATE_STATUS[col.status] ?? col.label}
                <span className="rounded-full bg-background px-2 text-xs text-muted-foreground tabular">{col.count}</span>
              </header>
              {col.candidates.length === 0 ? (
                <p className="px-1.5 py-6 text-center text-xs text-muted-foreground">{over === col.status ? rs.pipeline.dropHere : rs.pipeline.emptyColumn}</p>
              ) : (
                col.candidates.map((cand) => <CandidateCard key={cand.id} candidate={cand} canWrite={canWrite} />)
              )}
            </section>
          ))}
        </div>
      </div>
    </div>
  );
}

function CandidateCard({ candidate, canWrite }: { candidate: Candidate; canWrite: boolean }) {
  return (
    <article
      draggable={canWrite}
      onDragStart={(e) => {
        e.dataTransfer.setData("text/plain", candidate.id);
        e.dataTransfer.effectAllowed = "move";
      }}
      className={cn("rounded-lg border bg-card p-3 text-sm shadow-xs", canWrite && "cursor-grab active:cursor-grabbing")}
    >
      <Link href={`/erp/hr/recruitment/candidates/${candidate.id}`} className="flex items-center gap-1.5 font-medium hover:underline">
        <span className="truncate">{candidate.fullName}</span>
        {candidate.hasCv ? <FileText className="size-3.5 shrink-0 text-muted-foreground" aria-label={rs.profile.cv} /> : null}
      </Link>
      <p className="mt-0.5 truncate text-xs text-muted-foreground">{candidate.jobOpeningTitle || rs.candidates.noJob}</p>
      <div className="mt-2 flex items-center justify-between gap-2 text-[11px] text-muted-foreground">
        <span className="truncate">{SOURCE[candidate.source] ?? candidate.source}</span>
        <span className="tabular">{formatDate(candidate.updatedAt)}</span>
      </div>
      {canWrite ? (
        <div className="mt-2 md:hidden">
          <StageMenu candidate={candidate} size="xs" />
        </div>
      ) : null}
    </article>
  );
}
