import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { getCompanyProfile, getMailStatus } from "@/features/admin/company/data";
import { CompanyForm, MailForm } from "@/features/admin/company/forms";
import { tc } from "@/features/admin/company/strings";
import { load } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { requireAdmin } from "@/lib/rbac";
import { t } from "@/i18n/id";

export const metadata: Metadata = { title: tc.title };

export default async function CompanyPage() {
  await requireAdmin();
  const [profile, mail] = await Promise.all([load(getCompanyProfile), load(getMailStatus)]);
  return (
    <>
      <PageHeader eyebrow={t.nav.admin} title={tc.title} description={tc.subtitle} />
      <div className="grid gap-4 lg:grid-cols-[3fr_2fr] lg:items-start">
        {profile.error ? <ErpLoadError error={profile.error} /> : <CompanyForm profile={profile.data} />}
        {mail.error ? <ErpLoadError error={mail.error} /> : <MailForm status={mail.data} />}
      </div>
    </>
  );
}
