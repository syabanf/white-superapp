import type { Metadata } from "next";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/hr/components/ui";
import { listEmployeeOptions } from "@/features/erp/hr/queries";
import { AiInterviewPanel } from "@/features/erp/recruitment/components/ai-interview-panel";
import { EditCandidateButton } from "@/features/erp/recruitment/components/candidate-form";
import { DeleteCandidateButton, StageMenu } from "@/features/erp/recruitment/components/candidate-header";
import { ProfilePanel, Timeline } from "@/features/erp/recruitment/components/candidate-profile";
import { InterviewList, ScheduleButton } from "@/features/erp/recruitment/components/interviews";
import { OffersPanel } from "@/features/erp/recruitment/components/offers-panel";
import { PsikotesPanel } from "@/features/erp/recruitment/components/psikotes-panel";
import { ScreeningForm } from "@/features/erp/recruitment/components/screening-form";
import { CandidateBadge } from "@/features/erp/recruitment/components/ui";
import { REPORT_STAGES, recPerms } from "@/features/erp/recruitment/lib";
import {
  getAiAnalysis,
  getCandidate,
  getCandidateAiInterviews,
  getCandidateOffers,
  getCandidatePsikotes,
  getScreening,
  listActivities,
  listInstruments,
  listInterviews,
  listJobOpenings,
  listNotes,
} from "@/features/erp/recruitment/queries";
import { rs } from "@/features/erp/recruitment/strings";
import { initials } from "@/lib/format";

export const metadata: Metadata = { title: rs.candidates.title };

const TABS = Object.entries(rs.candidates.tabs).map(([value, label]) => ({ value, label }));
const none = { data: [] as never[], error: null };

export default async function CandidateDetailPage(props: PageProps<"/erp/hr/recruitment/candidates/[id]">) {
  const [{ id }, sp] = await Promise.all([props.params, props.searchParams]);
  const { access, offline } = await requireErp("recruitment.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const perms = recPerms(access);
  const tab = TABS.some((t) => t.value === sp1(sp.tab)) ? sp1(sp.tab)! : "profile";

  const [cand, jobs] = await Promise.all([
    load(() => getCandidate(id)),
    perms.write ? load(() => listJobOpenings()) : Promise.resolve(none),
  ]);
  if (!cand.data) return <ErpLoadError error={cand.error} />;
  const c = cand.data;
  const jobOptions = (jobs.data ?? []).map((j) => ({ value: j.id, label: j.title, hint: j.department }));

  return (
    <>
      <Button variant="ghost" size="sm" asChild className="-ml-2 self-start">
        <Link href="/erp/hr/recruitment/candidates">
          <ArrowLeft className="size-4" /> {rs.candidates.back}
        </Link>
      </Button>
      <PageHeader
        eyebrow={c.jobOpeningTitle || rs.candidates.noJob}
        title={
          <span className="flex items-center gap-3">
            <Avatar className="size-10">
              <AvatarFallback className="text-sm font-semibold">{initials(c.fullName)}</AvatarFallback>
            </Avatar>
            <span className="truncate">{c.fullName}</span>
          </span>
        }
        description={[c.email, c.phone].filter(Boolean).join(" · ")}
        actions={
          <>
            <CandidateBadge status={c.status} />
            {REPORT_STAGES.includes(c.status) ? (
              <DownloadButton href={`/api/erp/v1/recruitment/candidates/${c.id}/report`} fileName={`laporan-${c.id}.pdf`}>
                {rs.candidates.report}
              </DownloadButton>
            ) : null}
            {perms.write ? (
              <>
                <StageMenu candidate={c} />
                <EditCandidateButton candidate={c} jobs={jobOptions} />
                <DeleteCandidateButton id={c.id} />
              </>
            ) : null}
          </>
        }
      />
      <UrlTabs tabs={TABS} value={tab} />
      <CandidateTab tab={tab} id={id} candidate={c} canWrite={perms.write} />
    </>
  );
}

async function CandidateTab({
  tab,
  id,
  candidate,
  canWrite,
}: {
  tab: string;
  id: string;
  candidate: Awaited<ReturnType<typeof getCandidate>>;
  canWrite: boolean;
}) {
  switch (tab) {
    case "screening": {
      const res = await load(() => getScreening(id));
      if (res.error) return <ErpLoadError error={res.error} />;
      return <ScreeningForm candidateId={id} screening={res.data} canWrite={canWrite} />;
    }
    case "psikotes": {
      const [panel, instruments] = await Promise.all([
        load(() => getCandidatePsikotes(id)),
        canWrite ? load(listInstruments) : Promise.resolve(none),
      ]);
      if (!panel.data) return <ErpLoadError error={panel.error} />;
      return <PsikotesPanel candidateId={id} panel={panel.data} instruments={instruments.data ?? []} canWrite={canWrite} />;
    }
    case "ai": {
      const res = await load(() => getCandidateAiInterviews(id));
      if (!res.data) return <ErpLoadError error={res.error} />;
      return <AiInterviewPanel candidateId={id} sessions={res.data} canWrite={canWrite} />;
    }
    case "interviews": {
      const [list, employees] = await Promise.all([
        load(() => listInterviews({ candidateId: id })),
        canWrite ? load(listEmployeeOptions) : Promise.resolve(none),
      ]);
      if (!list.data) return <ErpLoadError error={list.error} />;
      const lookups = {
        candidates: [{ value: id, label: candidate.fullName }],
        employees: (employees.data ?? []).map((e) => ({ value: e.id, label: e.label })),
      };
      return (
        <div className="flex flex-col gap-4">
          <div className="flex items-center justify-between gap-2">
            <h2 className="text-base font-semibold">{rs.interviews.title}</h2>
            {canWrite ? <ScheduleButton lookups={lookups} candidateId={id} /> : null}
          </div>
          <InterviewList interviews={list.data} canWrite={canWrite} lookups={lookups} showCandidate={false} />
        </div>
      );
    }
    case "offers": {
      const res = await load(() => getCandidateOffers(id));
      if (!res.data) return <ErpLoadError error={res.error} />;
      return <OffersPanel candidateId={id} panel={res.data} canWrite={canWrite} />;
    }
    case "activity": {
      const res = await load(() => listActivities(id));
      if (!res.data) return <ErpLoadError error={res.error} />;
      return (
        <Card>
          <CardContent>
            <Timeline items={res.data} />
          </CardContent>
        </Card>
      );
    }
    default: {
      const [analysis, notes, activities] = await Promise.all([
        load(() => getAiAnalysis(id)),
        load(() => listNotes(id)),
        load(() => listActivities(id)),
      ]);
      return (
        <ProfilePanel
          candidate={candidate}
          analysis={analysis.data ?? null}
          notes={notes.data ?? []}
          history={(activities.data ?? []).filter((a) => a.activityType === "status_change")}
          canWrite={canWrite}
        />
      );
    }
  }
}
