import type { Metadata } from "next";
import Link from "next/link";
import { BarChart3 } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { formatRupiah, parseMoney } from "@/features/erp/shared/money";
import { MonthFilter } from "@/features/erp/shared/period-picker";
import { ErpError } from "@/features/erp/shared/states";
import { ForecastTable, TargetsDialog } from "@/features/erp/sales/components/reports";
import { currentMonth, salesCaps } from "@/features/erp/sales/lib";
import { getForecast, listTargets } from "@/features/erp/sales/queries";
import { ownerOptions } from "@/features/erp/sales/refs";
import { ss } from "@/features/erp/sales/strings";

const r = ss.reports;
export const metadata: Metadata = { title: r.title };

/** Monthly forecast per salesperson and target editing (sales.approve). Analytics live on /erp/sales. */
export default async function SalesReportsPage(props: PageProps<"/erp/sales/reports">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const month = sp1(sp.month) ?? currentMonth();
  const [forecast, targets, owners] = await Promise.all([
    load(() => getForecast(month)),
    caps.approve ? load(() => listTargets(month)) : null,
    caps.approve ? ownerOptions() : Promise.resolve([]),
  ]);

  return (
    <>
      <PageHeader
        eyebrow={ss.eyebrow}
        title={r.title}
        description={r.subtitle}
        actions={
          <Button size="sm" variant="outline" asChild>
            <Link href="/erp/sales?tab=tim">
              <BarChart3 className="size-3.5" /> {r.toDashboard}
            </Link>
          </Button>
        }
      />
      <div className="flex flex-wrap items-center justify-between gap-2">
        <MonthFilter fallback={month} />
        {caps.approve && targets?.data ? <TargetsDialog month={month} targets={targets.data.targets} owners={owners} /> : null}
      </div>
      {forecast.data ? (
        <>
          {!forecast.data.companyTargetSet ? <p className="text-sm text-muted-foreground">{ss.dashboard.noTarget}</p> : null}
          {parseMoney(forecast.data.allocatedTargetValue) > 0 ? (
            <p className="text-xs text-muted-foreground">
              {r.allocated}: {formatRupiah(forecast.data.allocatedTargetValue)}
            </p>
          ) : null}
          <ForecastTable forecast={forecast.data} />
        </>
      ) : (
        <ErpError message={forecast.error?.message} />
      )}
    </>
  );
}
