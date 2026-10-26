import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { InstrumentsView } from "@/features/erp/recruitment/components/psikotes-admin";
import { recPerms } from "@/features/erp/recruitment/lib";
import { listInstruments } from "@/features/erp/recruitment/queries";
import { rs } from "@/features/erp/recruitment/strings";

export const metadata: Metadata = { title: rs.bank.title };

export default async function PsikotesBankPage() {
  const { access, offline } = await requireErp("recruitment.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const res = await load(listInstruments);
  return (
    <>
      <PageHeader eyebrow={rs.eyebrow} title={rs.bank.title} description={rs.bank.subtitle} />
      {res.data ? <InstrumentsView instruments={res.data} canWrite={recPerms(access).write} /> : <ErpLoadError error={res.error} />}
    </>
  );
}
