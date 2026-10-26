import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { CheckCircle2, ChevronLeft, FileIcon } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { load } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { Panel } from "@/features/erp/sales/components/blocks";
import { CreateClientDialog, OnboardingChecklist } from "@/features/erp/sales/components/onboarding";
import { CopyLinkButton, SalesBadge } from "@/features/erp/sales/components/ui";
import { fileSize, salesCaps, tokenFromPath } from "@/features/erp/sales/lib";
import { getOnboarding } from "@/features/erp/sales/queries";
import { ownerOptions } from "@/features/erp/sales/refs";
import { ONBOARDING_STATUS, ss } from "@/features/erp/sales/strings";
import { db } from "@/lib/db";
import { formatDateTime } from "@/lib/format";

const s = ss.onboarding;
export const metadata: Metadata = { title: s.title };

export default async function OnboardingPage(props: PageProps<"/erp/sales/onboarding/[id]">) {
  const { id } = await props.params;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const [res, owners] = await Promise.all([load(() => getOnboarding(id)), ownerOptions()]);
  if (res.error?.code === "NOT_FOUND") notFound();
  if (!res.data) return <ErpError message={res.error?.message} />;
  const o = res.data;
  const portal = o.portalPath ? `/onboarding/${tokenFromPath(o.portalPath)}` : "";
  const marketingClient = o.marketingClientId
    ? await db.client.findUnique({ where: { id: o.marketingClientId }, select: { slug: true, name: true } }).catch(() => null)
    : null;
  const brief = Object.entries(o.brief).filter(([, v]) => v);

  return (
    <>
      <Link href="/erp/sales/onboarding" className="-mb-3 inline-flex w-fit items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
        <ChevronLeft className="size-4" /> {s.title}
      </Link>
      <PageHeader
        eyebrow={s.title}
        title={o.clientName}
        description={`${s.started} ${formatDateTime(o.createdAt)}`}
        actions={
          <>
            <Button size="sm" variant="outline" asChild>
              <Link href={`/erp/sales/pipeline/${o.dealId}`}>{ss.pipeline.title}</Link>
            </Button>
            {o.marketingClientId ? null : caps.admin ? <CreateClientDialog onboarding={o} /> : null}
          </>
        }
      />
      <div className="flex flex-wrap items-center gap-3">
        <SalesBadge value={o.status} map={ONBOARDING_STATUS} />
        <Progress value={o.progress} className="max-w-xs flex-1" />
        <span className="text-sm font-medium tabular">{o.progress}%</span>
      </div>
      <div className="grid gap-4 lg:grid-cols-3">
        <div className="flex flex-col gap-4 lg:col-span-2">
          <Panel title={s.checklist} flush>
            <OnboardingChecklist items={o.items} owners={owners} canWrite={caps.write} />
          </Panel>
          <Panel title={s.brief}>
            {brief.length ? (
              <>
                <dl className="grid gap-3 sm:grid-cols-2">
                  {brief.map(([k, v]) => (
                    <div key={k} className="min-w-0">
                      <dt className="text-xs text-muted-foreground">{k.replace(/_/g, " ")}</dt>
                      <dd className="text-sm whitespace-pre-line break-words">{v}</dd>
                    </div>
                  ))}
                </dl>
                {o.briefSubmittedAt ? (
                  <p className="mt-3 text-xs text-muted-foreground">
                    {s.briefAt} {formatDateTime(o.briefSubmittedAt)}
                  </p>
                ) : null}
              </>
            ) : (
              <p className="text-sm text-muted-foreground">{s.briefEmpty}</p>
            )}
          </Panel>
        </div>
        <div className="flex flex-col gap-4">
          {portal ? (
            <Panel title={s.portal}>
              <p className="mb-3 text-xs text-muted-foreground">{s.portalHint}</p>
              <p className="mb-3 truncate font-mono text-xs">{portal}</p>
              <CopyLinkButton path={portal} />
            </Panel>
          ) : null}
          {o.marketingClientId ? (
            <Panel title={s.createClientTitle}>
              <p className="flex items-center gap-2 text-sm">
                <CheckCircle2 className="size-4 text-positive" /> {s.clientLinked}
              </p>
              {marketingClient ? (
                <Button size="sm" variant="outline" className="mt-3" asChild>
                  <Link href={`/clients/${marketingClient.slug}`}>
                    {s.openClient}: {marketingClient.name}
                  </Link>
                </Button>
              ) : null}
            </Panel>
          ) : null}
          <Panel title={s.assets} flush>
            {o.assets.length === 0 ? (
              <p className="px-5 py-4 text-sm text-muted-foreground">{s.assetsEmpty}</p>
            ) : (
              <ul className="divide-y">
                {o.assets.map((a) => (
                  <li key={a.id} className="flex items-center gap-3 px-5 py-2.5 text-sm">
                    <FileIcon className="size-4 shrink-0 text-muted-foreground" />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate">{a.fileName}</span>
                      <span className="text-xs text-muted-foreground">
                        {fileSize(a.sizeBytes)} · {formatDateTime(a.createdAt)}
                      </span>
                    </span>
                    <DownloadButton href={`/api/erp/api/sales/onboarding/assets/${a.id}`} fileName={a.fileName} size="xs" variant="ghost">
                      {ss.open}
                    </DownloadButton>
                  </li>
                ))}
              </ul>
            )}
          </Panel>
        </div>
      </div>
    </>
  );
}
