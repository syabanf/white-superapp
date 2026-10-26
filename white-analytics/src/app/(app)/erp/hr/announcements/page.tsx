import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { AnnouncementsAdmin } from "@/features/erp/hr/components/announcements";
import { hrPerms } from "@/features/erp/hr/perms";
import { listAnnouncements, listDepartments } from "@/features/erp/hr/queries";
import { th } from "@/features/erp/hr/strings";

export const metadata: Metadata = { title: th.announcements.title };

export default async function AnnouncementsPage(props: PageProps<"/erp/hr/announcements">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("hr.read");
  const n = th.announcements;
  const header = <PageHeader eyebrow={th.eyebrow} title={n.title} description={n.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const [list, deps] = await Promise.all([load(() => listAnnouncements(sp1(sp.status))), load(() => listDepartments())]);
  return (
    <>
      {header}
      {list.data ? (
        <AnnouncementsAdmin
          items={list.data.announcements}
          departments={deps.data ?? []}
          canWrite={hrPerms(access).write}
        />
      ) : (
        <ErpError message={list.error.message} />
      )}
    </>
  );
}
