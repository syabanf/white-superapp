import type { Metadata } from "next";
import { Card } from "@/components/ui/card";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { listAssets, listTrainings } from "@/features/erp/hr/workplace/queries";
import { tw } from "@/features/erp/hr/workplace/strings";
import { tm } from "@/features/erp/me/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { formatDate } from "@/lib/format";

export const metadata: Metadata = { title: tm.assets.title };

/** What the employee holds and the training they joined. The services scope both to the caller. */
export default async function MyAssetsPage() {
  const { offline } = await requireErp("self.service");
  const header = <PageHeader eyebrow={tm.eyebrow} title={tm.assets.title} description={tm.assets.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const [assets, trainings] = await Promise.all([load(() => listAssets()), load(() => listTrainings())]);
  return (
    <>
      {header}
      <div className="grid gap-4 md:grid-cols-2">
        <Card className="gap-2 p-(--card-pad)">
          <h2 className="text-sm font-semibold">{tw.assets.mine}</h2>
          {assets.data?.length ? (
            <ul className="divide-y text-sm">
              {assets.data.map((a) => (
                <li key={a.id} className="py-2">
                  <p className="font-medium">{a.name}</p>
                  <p className="text-muted-foreground">
                    {a.assetCode}
                    {a.serialNo ? ` · ${a.serialNo}` : ""}
                  </p>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-sm text-muted-foreground">{assets.error?.message ?? tw.assets.mineEmpty}</p>
          )}
        </Card>
        <Card className="gap-2 p-(--card-pad)">
          <h2 className="text-sm font-semibold">{tw.training.mine}</h2>
          {trainings.data?.length ? (
            <ul className="divide-y text-sm">
              {trainings.data.map((t) => (
                <li key={t.id} className="py-2">
                  <p className="font-medium">{t.title}</p>
                  <p className="text-muted-foreground">
                    {formatDate(t.startDate)} – {formatDate(t.endDate)} · {tw.training.statuses[t.status]}
                  </p>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-sm text-muted-foreground">{trainings.error?.message ?? tw.training.mineEmpty}</p>
          )}
        </Card>
      </div>
    </>
  );
}
