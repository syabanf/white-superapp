import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { ChevronLeft } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { Card } from "@/components/ui/card";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { Panel } from "@/features/erp/sales/components/blocks";
import { ContractActions, ContractEditor } from "@/features/erp/sales/components/contracts";
import { Markdown } from "@/features/erp/sales/components/markdown";
import { CopyLinkButton, Info, SalesBadge } from "@/features/erp/sales/components/ui";
import { salesCaps, signPagePath } from "@/features/erp/sales/lib";
import { getContract } from "@/features/erp/sales/queries";
import { CONTRACT_STATUS, CONTRACT_TYPE, lbl, ss } from "@/features/erp/sales/strings";
import { formatDate, formatDateTime } from "@/lib/format";

const s = ss.contracts;
export const metadata: Metadata = { title: s.title };

export default async function ContractPage(props: PageProps<"/erp/sales/contracts/[id]">) {
  const { id } = await props.params;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const res = await load(() => getContract(id));
  if (res.error?.code === "NOT_FOUND") notFound();
  if (!res.data) return <ErpError message={res.error?.message} />;
  const c = res.data;
  const editable = caps.write && c.status === "draft";
  const trail: [string, string | null][] = [
    [ss.createdAt, c.createdAt],
    [CONTRACT_STATUS.sent, c.sentAt],
    [CONTRACT_STATUS.signed, c.signedAt],
    [CONTRACT_STATUS.active, c.activatedAt],
    [CONTRACT_STATUS.ended, c.endedAt],
    [CONTRACT_STATUS.cancelled, c.cancelledAt],
  ];

  return (
    <>
      <Link href="/erp/sales/contracts" className="-mb-3 inline-flex w-fit items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
        <ChevronLeft className="size-4" /> {s.title}
      </Link>
      <PageHeader
        eyebrow={`${c.contractNumber} · ${lbl(CONTRACT_TYPE, c.type)}`}
        title={c.title}
        description={c.clientName}
        actions={<ContractActions contract={c} caps={caps} />}
      />
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2">
          {editable ? (
            <ContractEditor contract={c} />
          ) : (
            <Card className="px-6 py-6 md:px-10">
              <Markdown source={c.body} />
            </Card>
          )}
        </div>
        <div className="flex flex-col gap-4">
          <Panel title={ss.detail}>
            <div className="mb-3">
              <SalesBadge value={c.status} map={CONTRACT_STATUS} />
            </div>
            <dl className="grid grid-cols-2 gap-x-4 gap-y-3">
              <Info label={s.start}>{formatDate(c.startDate || null)}</Info>
              <Info label={s.end}>{formatDate(c.endDate || null)}</Info>
              <Info label={ss.pipeline.title}>
                <Link href={`/erp/sales/pipeline/${c.dealId}`} className="hover:underline">
                  {ss.detail}
                </Link>
              </Info>
              <Info label={ss.quotations.title}>
                <Link href={`/erp/sales/quotations/${c.quotationId}`} className="hover:underline">
                  {ss.detail}
                </Link>
              </Info>
              {c.signerName ? <Info label={s.signedBy(c.signerName)}>{c.signerIp ? `${s.signerIp}: ${c.signerIp}` : null}</Info> : null}
              {c.cancelReason ? <Info label={s.cancelReason}>{c.cancelReason}</Info> : null}
            </dl>
          </Panel>
          {c.signPath ? (
            <Panel title={s.signLink}>
              <p className="mb-3 truncate font-mono text-xs text-muted-foreground">{signPagePath(c)}</p>
              <CopyLinkButton path={signPagePath(c)} />
            </Panel>
          ) : null}
          <Panel title={s.timeline}>
            <ol className="space-y-2 text-sm">
              {trail
                .filter(([, ts]) => ts)
                .map(([label, ts]) => (
                  <li key={label} className="flex justify-between gap-2">
                    <span>{label}</span>
                    <span className="text-xs text-muted-foreground tabular">{formatDateTime(ts)}</span>
                  </li>
                ))}
            </ol>
          </Panel>
          {!editable && caps.write && c.status !== "draft" ? <p className="text-xs text-muted-foreground">{s.draftOnly}</p> : null}
        </div>
      </div>
    </>
  );
}
