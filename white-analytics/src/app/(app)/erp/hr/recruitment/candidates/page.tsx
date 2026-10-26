import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1, spInt } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { NewCandidateButton } from "@/features/erp/recruitment/components/candidate-form";
import { CandidatesTable } from "@/features/erp/recruitment/components/candidates-table";
import { recPerms } from "@/features/erp/recruitment/lib";
import { listCandidates, listJobOpenings } from "@/features/erp/recruitment/queries";
import { rs } from "@/features/erp/recruitment/strings";

export const metadata: Metadata = { title: rs.candidates.title };

export default async function CandidatesPage(props: PageProps<"/erp/hr/recruitment/candidates">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("recruitment.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const perms = recPerms(access);
  const page = spInt(sp.page, 1);
  const pageSize = spInt(sp.pageSize, 20);

  const [list, jobs] = await Promise.all([
    load(() =>
      listCandidates({
        status: sp1(sp.status),
        jobOpeningId: sp1(sp.job),
        search: sp1(sp.q),
        source: sp1(sp.source),
        dateFrom: sp1(sp.from),
        dateTo: sp1(sp.to),
        sort: sp1(sp.sort),
        page,
        limit: pageSize,
      }),
    ),
    load(() => listJobOpenings()),
  ]);
  const jobOptions = (jobs.data ?? []).map((j) => ({ value: j.id, label: j.title, hint: j.department }));

  return (
    <>
      <PageHeader
        eyebrow={rs.eyebrow}
        title={rs.candidates.title}
        description={rs.candidates.subtitle}
        actions={perms.write ? <NewCandidateButton jobs={jobOptions} /> : null}
      />
      {list.data ? (
        <CandidatesTable
          candidates={list.data.candidates}
          pagination={{ page: list.data.page || page, pageSize, total: list.data.total }}
          jobs={jobOptions}
        />
      ) : (
        <ErpLoadError error={list.error} />
      )}
    </>
  );
}
