import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { hrPerms, todayWib } from "@/features/erp/hr/perms";
import { th } from "@/features/erp/hr/strings";
import { listTrainings } from "@/features/erp/hr/workplace/queries";
import { tw } from "@/features/erp/hr/workplace/strings";
import { TrainingsAdmin } from "@/features/erp/hr/workplace/training";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";

export const metadata: Metadata = { title: tw.training.title };

export default async function TrainingPage() {
  const { access, offline } = await requireErp("hr.read");
  const header = <PageHeader eyebrow={th.eyebrow} title={tw.training.title} description={tw.training.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const res = await load(() => listTrainings());
  return (
    <>
      {header}
      {res.data ? <TrainingsAdmin trainings={res.data} canWrite={hrPerms(access).write} today={todayWib()} /> : <ErpError message={res.error.message} />}
    </>
  );
}
