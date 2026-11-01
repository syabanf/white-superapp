import type { Metadata } from "next";
import Link from "next/link";
import { LayoutDashboard } from "lucide-react";
import { Button } from "@/components/ui/button";
import { GuidePage } from "@/features/guide/components/guide-page";
import { requireUser } from "@/lib/rbac";
import { tg } from "@/features/guide/strings";

export const metadata: Metadata = { title: tg.erpTitle };

/** ERP tutorial. ERP pages open the matching section from the topbar help button. */
export default async function ErpGuidePage() {
  await requireUser();

  return (
    <GuidePage
      track="erp"
      title={tg.erpTitle}
      description={tg.erpSubtitle}
      action={
        <Button asChild size="sm" className="group/nudge">
          <Link href="/erp">
            <LayoutDashboard className="size-4" /> {tg.erpCta}
          </Link>
        </Button>
      }
    />
  );
}
