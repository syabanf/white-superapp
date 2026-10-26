"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Loader2, Sparkles, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Textarea } from "@/components/ui/textarea";
import { ConfirmAction } from "@/features/erp/hr/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { FileUpload } from "@/features/erp/shared/file-upload";
import { Money } from "@/features/erp/shared/money-display";
import { formatDateTime } from "@/lib/format";
import { addNoteAction, analyzeCandidateAction, deleteCvAction } from "../actions";
import { AVAILABILITY, SOURCE, rs } from "../strings";
import type { AiAnalysis, Candidate, CandidateActivity, CandidateNote } from "../types";
import { Chips, Info } from "./ui";

const p = rs.profile;
const c = rs.candidates;

export function ProfilePanel({
  candidate,
  analysis,
  notes,
  history,
  canWrite,
}: {
  candidate: Candidate;
  analysis: AiAnalysis | null;
  notes: CandidateNote[];
  history: CandidateActivity[];
  canWrite: boolean;
}) {
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
      <div className="flex flex-col gap-4 lg:col-span-2">
        <Card>
          <CardHeader>
            <CardTitle>{p.contact}</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid gap-4 sm:grid-cols-2">
              <Info label={c.email}>{candidate.email}</Info>
              <Info label={c.phone}>{candidate.phone}</Info>
              <Info label={c.domicile}>{candidate.domicile}</Info>
              <Info label={c.source}>{SOURCE[candidate.source] ?? candidate.source}</Info>
              <Info label={c.job}>{candidate.jobOpeningTitle}</Info>
              <Info label={c.availability}>{AVAILABILITY[candidate.availability] ?? candidate.availability}</Info>
              <Info label={c.lastExperience}>{candidate.lastExperience}</Info>
              <Info label={c.lastEducation}>{candidate.lastEducation}</Info>
              <Info label={c.expectedSalary}>
                {candidate.expectedSalary ? <Money value={candidate.expectedSalary} /> : null}
              </Info>
              <Info label={c.applied}>{formatDateTime(candidate.createdAt)}</Info>
              <Info label={c.notes} className="sm:col-span-2">
                {candidate.notes ? <span className="whitespace-pre-wrap">{candidate.notes}</span> : null}
              </Info>
            </dl>
          </CardContent>
        </Card>
        <AiAnalysisCard candidateId={candidate.id} analysis={analysis} canWrite={canWrite} />
        <NotesCard candidateId={candidate.id} notes={notes} canWrite={canWrite} />
      </div>
      <div className="flex flex-col gap-4">
        <CvCard candidate={candidate} canWrite={canWrite} />
        <Card>
          <CardHeader>
            <CardTitle>{p.history}</CardTitle>
          </CardHeader>
          <CardContent>
            <Timeline items={history} />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function CvCard({ candidate, canWrite }: { candidate: Candidate; canWrite: boolean }) {
  const router = useRouter();
  return (
    <Card>
      <CardHeader>
        <CardTitle>{p.cv}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {candidate.hasCv ? (
          <div className="flex flex-wrap items-center gap-2">
            <span className="min-w-0 flex-1 truncate text-sm">{candidate.cvFileName}</span>
            <DownloadButton href={`/api/erp/v1/recruitment/candidates/${candidate.id}/cv`} fileName={candidate.cvFileName} />
            {canWrite ? (
              <ConfirmAction
                title={p.cvDeleteTitle}
                confirmLabel={rs.delete}
                success={p.cvDeleted}
                action={() => deleteCvAction(candidate.id)}
                trigger={
                  <Button size="icon-sm" variant="ghost" aria-label={p.cvDelete}>
                    <Trash2 className="size-4" />
                  </Button>
                }
              />
            ) : null}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">{p.noCv}</p>
        )}
        {canWrite ? (
          <FileUpload
            url={`/api/erp/api/recruitment/candidates/${candidate.id}/cv`}
            accept=".pdf,.doc,.docx,.jpg,.jpeg,.png"
            acceptLabel="PDF, DOC, DOCX, JPG, PNG"
            maxMb={10}
            multiple={false}
            onUploaded={() => router.refresh()}
          />
        ) : null}
      </CardContent>
    </Card>
  );
}

function AiAnalysisCard({ candidateId, analysis, canWrite }: { candidateId: string; analysis: AiAnalysis | null; canWrite: boolean }) {
  const { run, pending } = useErpAction();
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-2">
        <CardTitle>{p.ai}</CardTitle>
        {canWrite ? (
          <Button size="sm" variant="outline" disabled={pending} onClick={() => void run(() => analyzeCandidateAction(candidateId), { success: p.analyzed })}>
            {pending ? <Loader2 className="size-3.5 animate-spin" /> : <Sparkles className="size-3.5" />}
            {analysis ? p.reanalyze : p.analyze}
          </Button>
        ) : null}
      </CardHeader>
      <CardContent>
        {!analysis ? (
          <p className="text-sm text-muted-foreground">{p.noAnalysis}</p>
        ) : (
          <div className="flex flex-col gap-4 text-sm">
            <div>
              <div className="mb-1.5 flex items-baseline justify-between">
                <span className="text-xs text-muted-foreground">{p.matchScore}</span>
                <span className="text-lg font-semibold tabular">{analysis.matchScore}/100</span>
              </div>
              <Progress value={analysis.matchScore} />
              {analysis.matchReason ? <p className="mt-2 text-muted-foreground">{analysis.matchReason}</p> : null}
            </div>
            {analysis.summary ? <p className="whitespace-pre-wrap">{analysis.summary}</p> : null}
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <div className="mb-1.5 text-xs text-muted-foreground">{p.strengths}</div>
                <Chips items={analysis.strengths} />
              </div>
              <div>
                <div className="mb-1.5 text-xs text-muted-foreground">{p.gaps}</div>
                <Chips items={analysis.gaps} />
              </div>
            </div>
            <p className="text-xs text-muted-foreground">{p.model(analysis.model, formatDateTime(analysis.updatedAt))}</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function NotesCard({ candidateId, notes, canWrite }: { candidateId: string; notes: CandidateNote[]; canWrite: boolean }) {
  const [text, setText] = React.useState("");
  const { run, pending } = useErpAction();
  return (
    <Card>
      <CardHeader>
        <CardTitle>{p.notes}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        {canWrite ? (
          <form
            className="flex flex-col gap-2"
            onSubmit={(e) => {
              e.preventDefault();
              if (!text.trim()) return;
              void run(() => addNoteAction(candidateId, text), { success: p.noteAdded, onOk: () => setText("") });
            }}
          >
            <Textarea rows={2} value={text} onChange={(e) => setText(e.target.value)} placeholder={p.notePlaceholder} aria-label={p.addNote} />
            <Button type="submit" size="sm" className="self-end" disabled={pending || !text.trim()}>
              {pending ? <Loader2 className="size-3.5 animate-spin" /> : null}
              {p.addNote}
            </Button>
          </form>
        ) : null}
        {notes.length === 0 ? (
          <p className="text-sm text-muted-foreground">{p.noNotes}</p>
        ) : (
          <ul className="flex flex-col gap-3">
            {notes.map((n) => (
              <li key={n.id} className="rounded-lg bg-muted/50 px-3 py-2 text-sm">
                <p className="whitespace-pre-wrap">{n.content}</p>
                <p className="mt-1 text-xs text-muted-foreground">
                  {n.createdByName || rs.none} · {formatDateTime(n.createdAt)}
                </p>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}

/** Vertical list of activities, newest first. */
export function Timeline({ items }: { items: CandidateActivity[] }) {
  if (!items.length) return <p className="text-sm text-muted-foreground">{p.noActivity}</p>;
  return (
    <ol className="relative flex flex-col gap-4 border-l pl-4">
      {items.map((a) => (
        <li key={a.id} className="relative text-sm">
          <span className="absolute top-1.5 -left-[21px] size-2.5 rounded-full border-2 border-background bg-brand" />
          <p>{a.description}</p>
          <p className="text-xs text-muted-foreground">
            {[a.createdByName, formatDateTime(a.createdAt)].filter(Boolean).join(" · ")}
          </p>
        </li>
      ))}
    </ol>
  );
}
