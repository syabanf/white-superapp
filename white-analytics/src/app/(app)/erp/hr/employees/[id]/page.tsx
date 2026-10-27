import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { ContractsTable } from "@/features/erp/hr/components/contracts";
import { HistoryList, ProfileData } from "@/features/erp/hr/components/profile-data";
import {
  BalancePanel,
  DocumentsPanel,
  LifecyclePanel,
  ProfileActions,
  SchedulePanel,
} from "@/features/erp/hr/components/profile-panels";
import { UrlTabs } from "@/features/erp/hr/components/ui";
import { hrPerms } from "@/features/erp/hr/perms";
import {
  getEmployeeProfile,
  getEmployeeSchedule,
  getLeaveBalance,
  getOffboarding,
  getOnboarding,
  listContracts,
  listEmployeeDocuments,
  listEmploymentHistory,
  listShifts,
  listUserOptions,
} from "@/features/erp/hr/queries";
import { employeeStatusLabels, kinds, th } from "@/features/erp/hr/strings";
import { initials } from "@/lib/format";

export const metadata: Metadata = { title: th.employees.title };

const TABS = Object.entries(th.employees.profileTabs).map(([value, label]) => ({ value, label }));

export default async function EmployeeProfilePage(props: PageProps<"/erp/hr/employees/[id]">) {
  const [{ id }, sp] = await Promise.all([props.params, props.searchParams]);
  const { access, offline } = await requireErp("hr.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const perms = hrPerms(access);
  const tab = TABS.some((t) => t.value === sp1(sp.tab)) ? sp1(sp.tab)! : "data";

  const [profile, users] = await Promise.all([
    load(() => getEmployeeProfile(id)),
    perms.write ? load(listUserOptions) : Promise.resolve({ data: [], error: null }),
  ]);
  if (!profile.data) return <ErpError message={profile.error.message} />;
  const p = profile.data;
  const e = p.employee;

  return (
    <>
      <PageHeader
        back={{ href: "/erp/hr/employees", label: th.employees.back }}
        eyebrow={`${e.employeeNo} · ${p.employmentStatusName || "–"}`}
        title={
          <span className="flex items-center gap-3">
            <Avatar className="size-10">
              <AvatarFallback className="text-sm font-semibold">{initials(e.fullName)}</AvatarFallback>
            </Avatar>
            <span className="truncate">{e.fullName}</span>
          </span>
        }
        description={[e.positionName, e.departmentName].filter(Boolean).join(" · ")}
        actions={
          <>
            <ErpStatusBadge status={e.status} labels={employeeStatusLabels} kinds={kinds.employee} />
            {perms.write ? <ProfileActions profile={p} users={users.data ?? []} /> : null}
          </>
        }
      />
      <UrlTabs tabs={TABS} value={tab} />
      <ProfileTab tab={tab} id={id} canWrite={perms.write} profileData={p} />
    </>
  );
}

async function ProfileTab({
  tab,
  id,
  canWrite,
  profileData,
}: {
  tab: string;
  id: string;
  canWrite: boolean;
  profileData: Awaited<ReturnType<typeof getEmployeeProfile>>;
}) {
  switch (tab) {
    case "history": {
      const res = await load(() => listEmploymentHistory(id));
      return res.data ? <HistoryList entries={res.data} /> : <ErpError message={res.error.message} />;
    }
    case "documents": {
      const res = await load(() => listEmployeeDocuments(id));
      return res.data ? (
        <DocumentsPanel employeeId={id} documents={res.data} canWrite={canWrite} />
      ) : (
        <ErpError message={res.error.message} />
      );
    }
    case "schedule": {
      const [sched, shifts] = await Promise.all([load(() => getEmployeeSchedule(id)), load(() => listShifts())]);
      return sched.data ? (
        <SchedulePanel schedule={sched.data} shifts={shifts.data ?? []} canWrite={canWrite} />
      ) : (
        <ErpError message={sched.error.message} />
      );
    }
    case "leave": {
      const res = await load(() => getLeaveBalance(id));
      return res.data ? <BalancePanel balance={res.data} canWrite={canWrite} /> : <ErpError message={res.error.message} />;
    }
    case "contracts": {
      const res = await load(() => listContracts({ employeeId: id, status: "all" }));
      if (!res.data) return <ErpError message={res.error.message} />;
      const self = [{ id, label: `${profileData.employee.fullName} · ${profileData.employee.employeeNo}` }];
      return (
        <ContractsTable contracts={res.data} employees={self} defaultEmployeeId={id} canWrite={canWrite} filters={false} />
      );
    }
    case "lifecycle": {
      const [on, off] = await Promise.all([load(() => getOnboarding(id)), load(() => getOffboarding(id))]);
      if (!on.data) return <ErpError message={on.error.message} />;
      // NOT_FOUND means no offboarding has started yet.
      if (off.error && off.error.code !== "NOT_FOUND") return <ErpError message={off.error.message} />;
      return <LifecyclePanel employeeId={id} onboarding={on.data} offboarding={off.data} canWrite={canWrite} />;
    }
    default:
      return <ProfileData p={profileData} />;
  }
}
