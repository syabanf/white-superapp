import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { PipelineBoard } from "@/features/erp/recruitment/components/pipeline-board";
import { recPerms } from "@/features/erp/recruitment/lib";
import { getPipeline, listJobOpenings } from "@/features/erp/recruitment/queries";
import { rs } from "@/features/erp/recruitment/strings";

export const metadata: Metadata = { title: rs.pipeline.title };

export default async function PipelinePage(props: PageProps<"/erp/hr/recruitment/pipeline">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("recruitment.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const perms = recPerms(access);
  const [board, jobs] = await Promise.all([
    load(() => getPipeline({ jobOpeningId: sp1(sp.job), search: sp1(sp.q) })),
    load(() => listJobOpenings()),
  ]);

  return (
    <>
      <PageHeader eyebrow={rs.eyebrow} title={rs.pipeline.title} description={rs.pipeline.subtitle} />
      {board.data ? (
        <PipelineBoard
          columns={board.data.columns}
          jobs={(jobs.data ?? []).map((j) => ({ value: j.id, label: j.title, hint: j.department }))}
          canWrite={perms.write}
        />
      ) : (
        <ErpLoadError error={board.error} />
      )}
    </>
  );
}
