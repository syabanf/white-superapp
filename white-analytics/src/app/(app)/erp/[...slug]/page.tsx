import Link from "next/link";
import { Construction } from "lucide-react";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/dashboard/empty-state";
import { requireErp } from "@/features/erp/access";
import { t } from "@/i18n/id";

/**
 * Placeholder for ERP routes whose pages have not landed yet. Each ported
 * page is a concrete route and takes precedence over this catch-all.
 */
/** Permission each ERP area needs, by URL prefix (most specific first). */
const AREA_PERMS: [string, string][] = [
  ["hr/payroll", "hr.payroll"],
  ["hr/recruitment", "recruitment.read"],
  ["finance", "finance.read"],
  ["hr", "hr.read"],
  ["performance", "hr.read"],
  ["sales", "sales.read"],
  ["me", "self.service"],
];

export default async function ErpComingSoonPage(props: PageProps<"/erp/[...slug]">) {
  const { slug } = await props.params;
  const path = slug.join("/");
  const perm = AREA_PERMS.find(([prefix]) => path === prefix || path.startsWith(prefix + "/"))?.[1];
  await requireErp(perm);
  return (
    <EmptyState
      icon={<Construction className="size-5" />}
      title={t.erp.comingTitle}
      description={t.erp.comingBody}
      action={
        <Button asChild size="sm" variant="secondary">
          <Link href="/erp">{t.erp.backHome}</Link>
        </Button>
      }
    />
  );
}
