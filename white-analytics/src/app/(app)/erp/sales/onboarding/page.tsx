import type { Metadata } from "next";
import Link from "next/link";
import { PageHeader } from "@/components/dashboard/page-header";
import { Card } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpEmpty, ErpError } from "@/features/erp/shared/states";
import { SelectFilter } from "@/features/erp/shared/url-filters";
import { SalesBadge } from "@/features/erp/sales/components/ui";
import { listOnboardings } from "@/features/erp/sales/queries";
import { ONBOARDING_STATUS, opts, ss } from "@/features/erp/sales/strings";
import { formatDate } from "@/lib/format";

const s = ss.onboarding;
export const metadata: Metadata = { title: s.title };

export default async function OnboardingListPage(props: PageProps<"/erp/sales/onboarding">) {
  const sp = await props.searchParams;
  const { offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const status = sp1(sp.status);
  const list = await load(() => listOnboardings(status));

  return (
    <>
      <PageHeader eyebrow={ss.eyebrow} title={s.title} description={s.subtitle} />
      <div>
        <SelectFilter param="status" label={s.status} allLabel={s.allStatus} options={opts(ONBOARDING_STATUS)} />
      </div>
      {!list.data ? (
        <ErpError message={list.error?.message} />
      ) : list.data.length === 0 ? (
        <ErpEmpty description={s.empty} filtered={!!status} />
      ) : (
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {list.data.map((o) => {
            const open = o.items.filter((i) => i.status === "todo").length;
            return (
              <Link key={o.id} href={`/erp/sales/onboarding/${o.id}`} className="group">
                <Card className="h-full gap-3 px-5 py-4 transition-shadow group-hover:shadow-[var(--lift-shadow)]">
                  <div className="flex items-start justify-between gap-2">
                    <h2 className="font-medium">{o.clientName}</h2>
                    <SalesBadge value={o.status} map={ONBOARDING_STATUS} />
                  </div>
                  <div className="flex items-center gap-3">
                    <Progress value={o.progress} className="flex-1" />
                    <span className="text-sm font-medium tabular">{o.progress}%</span>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    {s.started} {formatDate(o.createdAt)} · {open} {s.todo.toLowerCase()}
                    {o.briefSubmittedAt ? ` · ${s.brief}` : ""}
                    {o.assets.length ? ` · ${o.assets.length} ${s.assets.toLowerCase()}` : ""}
                  </p>
                </Card>
              </Link>
            );
          })}
        </div>
      )}
    </>
  );
}
