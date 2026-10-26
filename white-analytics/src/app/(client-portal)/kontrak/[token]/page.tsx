import type { Metadata } from "next";
import { Card } from "@/components/ui/card";
import { Markdown } from "@/features/erp/sales/components/markdown";
import { loadPublic } from "@/features/erp/sales/public/load";
import { SignContract } from "@/features/erp/sales/public/sign-contract";
import { PublicProblem } from "@/features/erp/sales/public/states";
import { ps } from "@/features/erp/sales/strings";
import type { PublicContract } from "@/features/erp/sales/types";
import { formatDate } from "@/lib/format";

export const metadata: Metadata = { title: ps.contract.eyebrow };

export default async function PublicContractPage(props: PageProps<"/kontrak/[token]">) {
  const { token } = await props.params;
  const res = await loadPublic<PublicContract>(`/v1/public/sales/contracts/${encodeURIComponent(token)}`);
  if (!res.data) return <PublicProblem kind={res.error} message={res.message} />;
  const c = res.data;
  return (
    <>
      <div>
        <p className="label-mono text-muted-foreground">
          {ps.contract.eyebrow} · {c.contractNumber}
        </p>
        <h1 className="mt-1 text-[26px] leading-tight font-semibold">{c.title}</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {c.clientName}
          {c.startDate ? ` · ${ps.contract.period} ${formatDate(c.startDate)} – ${formatDate(c.endDate || null)}` : ""}
        </p>
      </div>
      <Card className="px-6 py-8 md:px-12">
        <Markdown source={c.body} />
      </Card>
      <SignContract token={token} contract={c} />
    </>
  );
}
