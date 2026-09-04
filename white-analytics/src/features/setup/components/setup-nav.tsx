import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { WizardNav } from "@/features/clients/components/wizard/wizard-nav";
import { nextSetupStep, prevSetupStep, setupHref, type SetupStep } from "@/features/setup/steps";
import { skipSetupAction } from "@/features/setup/actions";
import { ts } from "@/features/setup/strings";

/** Footer for setup steps: back / skip-all / next. Saving happens per card, so "Lanjut" is a plain link. */
export function SetupNav({ step }: { step: SetupStep }) {
  const prev = prevSetupStep(step);
  const next = nextSetupStep(step);
  return (
    <WizardNav
      backHref={prev ? { href: setupHref(prev), label: ts.back } : undefined}
      secondary={
        <form action={skipSetupAction}>
          <Button type="submit" variant="ghost">
            {ts.skipAll}
          </Button>
        </form>
      }
      submit={
        next ? (
          <Button asChild className="group/nudge">
            <Link href={setupHref(next)}>
              {ts.next} <ArrowRight className="nudge nudge-x size-4" />
            </Link>
          </Button>
        ) : null
      }
    />
  );
}
