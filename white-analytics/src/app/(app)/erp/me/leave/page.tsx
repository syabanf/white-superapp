import type { Metadata } from "next";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { BalancePanel } from "@/features/erp/hr/components/profile-panels";
import { listLeaveTypes } from "@/features/erp/hr/queries";
import { MyLeaveTable } from "@/features/erp/me/components/lists";
import { getMyLeaveBalance, listMyLeaves } from "@/features/erp/me/queries";
import { tm } from "@/features/erp/me/strings";

export const metadata: Metadata = { title: tm.leave.title };

export default async function MyLeavePage() {
  const { offline } = await requireErp("self.service");
  const header = <PageHeader eyebrow={tm.eyebrow} title={tm.leave.title} description={tm.leave.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const year = new Date().getFullYear();
  const [balance, leaves, types] = await Promise.all([
    load(() => getMyLeaveBalance(year)),
    load(() => listMyLeaves(year)),
    load(listLeaveTypes),
  ]);
  if (!balance.data) return <>{header}<ErpError message={balance.error.message} /></>;
  return (
    <>
      {header}
      <BalancePanel balance={balance.data} canWrite={false} />
      <Section title={tm.leave.history}>
        {leaves.data ? (
          <MyLeaveTable leaves={leaves.data} types={types.data ?? []} balance={balance.data} />
        ) : (
          <ErpError message={leaves.error.message} />
        )}
      </Section>
    </>
  );
}
