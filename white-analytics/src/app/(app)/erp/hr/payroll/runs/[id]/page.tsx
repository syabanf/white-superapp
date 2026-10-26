import type { Metadata } from "next";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { KpiTile } from "@/components/dashboard/kpi-tile";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { formatRupiah } from "@/features/erp/shared/money";
import { ErpError } from "@/features/erp/shared/states";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { PayslipsTable, RunActions } from "@/features/erp/hr/components/payroll-runs";
import { hrPerms } from "@/features/erp/hr/perms";
import { getPayrollRun } from "@/features/erp/hr/queries";
import { kinds, monthLabel, th } from "@/features/erp/hr/strings";
import { formatNumber } from "@/lib/format";

export const metadata: Metadata = { title: th.payroll.title };

export default async function PayrollRunPage(props: PageProps<"/erp/hr/payroll/runs/[id]">) {
  const { id } = await props.params;
  const { access, offline } = await requireErp("hr.payroll");
  if (offline) return <OfflineNotice message={offline.message} />;
  const p = th.payroll;
  const res = await load(() => getPayrollRun(id));
  if (!res.data) return <ErpError message={res.error.message} />;
  const { run, payslips } = res.data;

  return (
    <>
      <Button variant="ghost" size="sm" asChild className="-ml-2 self-start">
        <Link href="/erp/hr/payroll">
          <ArrowLeft className="size-4" /> {p.backToRuns}
        </Link>
      </Button>
      <PageHeader
        eyebrow={monthLabel(run.periodMonth, run.periodYear)}
        title={run.runName}
        actions={<ErpStatusBadge status={run.status} labels={p.statuses} kinds={kinds.payroll} />}
      />
      <Card className="p-4 sm:p-5">
        <RunActions run={run} canWrite={hrPerms(access).payroll} />
      </Card>
      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        <KpiTile label={p.employees} value={formatNumber(run.totalEmployees)} size="sm" />
        <KpiTile label={p.gross} value={formatRupiah(run.totalGross)} size="sm" />
        <KpiTile label={p.deductions} value={formatRupiah(run.totalDeductions)} size="sm" />
        <KpiTile label={p.net} value={formatRupiah(run.totalNet)} size="sm" />
        <KpiTile label={p.bpjsEmployee} value={formatRupiah(run.totalBpjsEmployee)} size="sm" />
        <KpiTile label={p.bpjsEmployer} value={formatRupiah(run.totalBpjsEmployer)} size="sm" />
        <KpiTile label={p.pph21} value={formatRupiah(run.totalPph21)} size="sm" />
      </div>
      <Section title={p.payslips}>
        <PayslipsTable payslips={payslips} />
      </Section>
    </>
  );
}
