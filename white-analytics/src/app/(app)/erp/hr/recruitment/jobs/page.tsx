import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { listDepartments, listUserOptions } from "@/features/erp/hr/queries";
import { JobFormButton, JobOpeningsView } from "@/features/erp/recruitment/components/job-openings";
import { recPerms } from "@/features/erp/recruitment/lib";
import { listJobOpenings } from "@/features/erp/recruitment/queries";
import { rs } from "@/features/erp/recruitment/strings";

export const metadata: Metadata = { title: rs.jobs.title };

export default async function JobOpeningsPage(props: PageProps<"/erp/hr/recruitment/jobs">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("recruitment.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const perms = recPerms(access);
  const status = sp1(sp.status);
  const search = sp1(sp.q);

  const [jobs, departments, users] = await Promise.all([
    load(() => listJobOpenings({ status, search })),
    perms.write ? load(() => listDepartments()) : Promise.resolve({ data: [], error: null }),
    perms.write ? load(listUserOptions) : Promise.resolve({ data: [], error: null }),
  ]);
  const lookups = {
    departments: (departments.data ?? []).map((d) => ({ value: d.id, label: d.name })),
    users: (users.data ?? []).map((u) => ({ value: u.id, label: u.name, hint: u.email })),
  };

  return (
    <>
      <PageHeader
        eyebrow={rs.eyebrow}
        title={rs.jobs.title}
        description={rs.jobs.subtitle}
        actions={perms.write ? <JobFormButton {...lookups} /> : null}
      />
      {jobs.data ? (
        <JobOpeningsView jobs={jobs.data} canWrite={perms.write} filtered={Boolean(status || search)} {...lookups} />
      ) : (
        <ErpLoadError error={jobs.error} />
      )}
    </>
  );
}
