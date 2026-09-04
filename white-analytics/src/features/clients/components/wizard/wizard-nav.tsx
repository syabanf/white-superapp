import type { ReactNode } from "react";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { wizardHref, type ProjectType, type WizardStep } from "@/features/clients/wizard";
import { tc } from "@/features/clients/strings";

/**
 * Shared footer row for wizard steps: back on the left, primary on the right.
 * On phones it sticks to the bottom of the viewport (with safe-area padding)
 * so the primary action is always one thumb away; from `sm` up it is a plain
 * footer inside the panel.
 */
export function WizardNav({
  slug,
  types,
  back,
  backHref,
  submit,
  secondary,
}: {
  slug?: string;
  types?: readonly ProjectType[];
  back?: WizardStep | null;
  /** explicit back link when there is no previous step (e.g. cancel) */
  backHref?: { href: string; label: string };
  submit: ReactNode;
  secondary?: ReactNode;
}) {
  const backLink = back && (slug || types?.length) ? { href: wizardHref(back, { slug, types }), label: tc.wizard.back } : backHref;
  return (
    <div className="sticky bottom-0 z-20 -mx-4 mt-2 flex items-center justify-between gap-3 border-t bg-background/95 px-4 pt-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:static sm:mx-0 sm:mt-0 sm:border-t sm:bg-transparent sm:px-0 sm:pt-5 sm:pb-0 sm:backdrop-blur-none">
      {backLink ? (
        <Button asChild variant="ghost" type="button" className="group/nudge">
          <Link href={backLink.href}>
            <ArrowLeft className="nudge nudge-back size-4" /> {backLink.label}
          </Link>
        </Button>
      ) : (
        <span />
      )}
      <div className="flex items-center gap-2">
        {secondary}
        {submit}
      </div>
    </div>
  );
}
