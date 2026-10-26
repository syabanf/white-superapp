import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { ContractTemplates, ContractsTable } from "@/features/erp/sales/components/contracts";
import { salesCaps } from "@/features/erp/sales/lib";
import { listContractTemplates, listContracts, listQuotations } from "@/features/erp/sales/queries";
import { ss } from "@/features/erp/sales/strings";

const s = ss.contracts;
export const metadata: Metadata = { title: s.title };

export default async function ContractsPage(props: PageProps<"/erp/sales/contracts">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const tab = sp1(sp.tab) ?? "list";
  const templates = await load(listContractTemplates);

  let body: React.ReactNode;
  if (tab === "templates") {
    body = templates.data ? (
      <ContractTemplates templates={templates.data.templates} placeholders={templates.data.placeholders} admin={caps.admin} />
    ) : (
      <ErpError message={templates.error?.message} />
    );
  } else {
    const [contracts, accepted] = await Promise.all([
      load(() => listContracts({ status: sp1(sp.status), limit: 500 })),
      caps.write ? load(() => listQuotations({ status: "diterima", limit: 200 })) : null,
    ]);
    body = contracts.data ? (
      <ContractsTable
        contracts={contracts.data}
        caps={caps}
        quotations={accepted?.data ?? []}
        templates={templates.data?.templates ?? []}
        generateFor={sp1(sp.generate)}
      />
    ) : (
      <ErpError message={contracts.error?.message} />
    );
  }

  return (
    <>
      <PageHeader eyebrow={ss.eyebrow} title={s.title} description={s.subtitle} />
      <UrlTabs
        label={s.title}
        tabs={[
          { value: "list", label: s.tabs.list },
          { value: "templates", label: s.tabs.templates },
        ]}
        reset={["status", "generate"]}
      />
      {body}
    </>
  );
}
