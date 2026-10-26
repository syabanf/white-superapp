import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { CaseStudies, DeckList, GeneratePitchDeckDialog } from "@/features/erp/sales/components/decks";
import { salesCaps } from "@/features/erp/sales/lib";
import { listCaseStudies, listDeals, listPitchDecks } from "@/features/erp/sales/queries";
import { ss } from "@/features/erp/sales/strings";

const s = ss.decks;
export const metadata: Metadata = { title: s.title };

export default async function PitchDecksPage(props: PageProps<"/erp/sales/pitch-decks">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const tab = sp1(sp.tab) ?? "decks";

  let body: React.ReactNode;
  let actions: React.ReactNode = null;
  if (tab === "cases") {
    const cases = await load(listCaseStudies);
    body = cases.data ? <CaseStudies items={cases.data} admin={caps.admin} /> : <ErpError message={cases.error?.message} />;
  } else {
    const [decks, deals] = await Promise.all([load(() => listPitchDecks()), load(() => listDeals({ pageSize: 500 }))]);
    const allDeals = deals.data?.deals ?? [];
    const dealNames = Object.fromEntries(allDeals.map((d) => [d.id, `${d.title}`]));
    const openDeals = allDeals.filter((d) => !d.isLost).map((d) => ({ value: d.id, label: `${d.title} · ${d.clientName}` }));
    if (caps.write) actions = <GeneratePitchDeckDialog deals={openDeals} />;
    body = decks.data ? <DeckList decks={decks.data} dealNames={dealNames} /> : <ErpError message={decks.error?.message} />;
  }

  return (
    <>
      <PageHeader eyebrow={ss.eyebrow} title={s.title} description={s.subtitle} actions={actions} />
      <UrlTabs
        label={s.title}
        tabs={[
          { value: "decks", label: s.tabs.decks },
          { value: "cases", label: s.tabs.cases },
        ]}
      />
      {body}
    </>
  );
}
