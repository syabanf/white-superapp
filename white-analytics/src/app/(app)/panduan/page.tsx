import type { Metadata } from "next";
import Link from "next/link";
import { Rocket } from "lucide-react";
import { Button } from "@/components/ui/button";
import { GuidePage } from "@/features/guide/components/guide-page";
import { requireUser } from "@/lib/rbac";
import { tg } from "@/features/guide/strings";

export const metadata: Metadata = { title: tg.title };

/** Digital Marketing tutorial. The topbar help button opens the same content, one section at a time. */
export default async function MarketingGuidePage() {
  await requireUser();

  return (
    <GuidePage
      track="marketing"
      title={tg.title}
      description={tg.subtitle}
      action={
        <Button asChild size="sm" className="group/nudge">
          <Link href="/clients/new">
            <Rocket className="size-4" /> {tg.startCta}
          </Link>
        </Button>
      }
    />
  );
}
