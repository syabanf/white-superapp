import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { hrPerms } from "@/features/erp/hr/perms";
import { listEmployeeOptions } from "@/features/erp/hr/queries";
import { th } from "@/features/erp/hr/strings";
import { getTraining } from "@/features/erp/hr/workplace/queries";
import { tw } from "@/features/erp/hr/workplace/strings";
import { TrainingParticipants } from "@/features/erp/hr/workplace/training";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { formatDate } from "@/lib/format";

export const metadata: Metadata = { title: tw.training.title };

export default async function TrainingDetailPage(props: PageProps<"/erp/hr/training/[id]">) {
  const { id } = await props.params;
  const { access, offline } = await requireErp("hr.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const [res, employees] = await Promise.all([load(() => getTraining(id)), load(listEmployeeOptions)]);
  if (!res.data) return <ErpError message={res.error.message} />;
  const t = res.data.training;
  const meta = [t.provider, `${formatDate(t.startDate)} – ${formatDate(t.endDate)}`, t.location, tw.training.statuses[t.status]].filter(Boolean).join(" · ");
  return (
    <>
      <PageHeader eyebrow={th.eyebrow} title={t.title} description={meta} />
      <TrainingParticipants training={t} participants={res.data.participants} employees={employees.data ?? []} canWrite={hrPerms(access).write} />
    </>
  );
}
