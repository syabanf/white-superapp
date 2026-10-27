import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpEmpty, ErpError } from "@/features/erp/shared/states";
import { FormsList, SubmissionsTable } from "@/features/erp/sales/components/forms";
import { salesCaps } from "@/features/erp/sales/lib";
import { listFormSubmissions, listForms } from "@/features/erp/sales/queries";
import { ownerOptions } from "@/features/erp/sales/refs";
import { ss } from "@/features/erp/sales/strings";

const f = ss.forms;
export const metadata: Metadata = { title: f.title };

export default async function FormsPage(props: PageProps<"/erp/sales/forms">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const [forms, owners] = await Promise.all([load(listForms), ownerOptions()]);
  if (!forms.data) return <ErpError message={forms.error?.message} />;
  const selected = forms.data.find((x) => x.id === sp1(sp.form));

  if (selected) {
    const subs = await load(() => listFormSubmissions(selected.id));
    return (
      <>
        <PageHeader
          back={{ href: "/erp/sales/forms", label: f.title }}
          eyebrow={f.submissions}
          title={selected.name}
          description={`/f/${selected.slug}`}
        />
        {subs.data ? (
          <SubmissionsTable form={selected} submissions={subs.data} />
        ) : (
          <ErpError message={subs.error?.message} />
        )}
      </>
    );
  }

  return (
    <>
      <PageHeader eyebrow={ss.eyebrow} title={f.title} description={f.subtitle} />
      {forms.data.length || caps.write ? (
        <FormsList forms={forms.data} owners={owners} canWrite={caps.write} />
      ) : (
        <ErpEmpty />
      )}
    </>
  );
}
