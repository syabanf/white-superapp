import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1, spInt } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { TalentPoolTable } from "@/features/erp/recruitment/components/talent-pool";
import { recPerms } from "@/features/erp/recruitment/lib";
import { listTalentPool } from "@/features/erp/recruitment/queries";
import { rs } from "@/features/erp/recruitment/strings";

export const metadata: Metadata = { title: rs.talentPool.title };

export default async function TalentPoolPage(props: PageProps<"/erp/hr/recruitment/talent-pool">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("recruitment.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const page = spInt(sp.page, 1);
  const pageSize = spInt(sp.pageSize, 20);
  const res = await load(() =>
    listTalentPool({ search: sp1(sp.q), includeArchived: sp1(sp.archived) === "1", page, limit: pageSize }),
  );

  return (
    <>
      <PageHeader eyebrow={rs.eyebrow} title={rs.talentPool.title} description={rs.talentPool.subtitle} />
      {res.data ? (
        <TalentPoolTable
          candidates={res.data.candidates}
          pagination={{ page: res.data.page || page, pageSize, total: res.data.total }}
          canWrite={recPerms(access).write}
        />
      ) : (
        <ErpLoadError error={res.error} />
      )}
    </>
  );
}
