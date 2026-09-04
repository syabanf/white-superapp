"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { ArrowUpRight, HelpCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { GuideBody } from "@/features/guide/components/guide-blocks";
import { getGuideSection, guideSectionForPath } from "@/features/guide/content";
import { tg } from "@/features/guide/strings";

/**
 * Contextual help. Opens the guide section that matches the page you are on,
 * with a link to the full guide — same content source, no duplicated copy.
 */
export function HelpButton() {
  const pathname = usePathname();
  const [open, setOpen] = React.useState(false);
  const section = getGuideSection(guideSectionForPath(pathname));
  if (!section) return null;

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <Tooltip>
        <TooltipTrigger asChild>
          <SheetTrigger asChild>
            <Button variant="ghost" size="icon-sm" aria-label={tg.helpAria}>
              <HelpCircle className="size-4" />
            </Button>
          </SheetTrigger>
        </TooltipTrigger>
        <TooltipContent>{tg.helpAria}</TooltipContent>
      </Tooltip>

      <SheetContent className="w-full gap-0 overflow-y-auto scrollbar-thin sm:max-w-md">
        <SheetHeader className="border-b">
          <p className="label-mono text-brand">{tg.eyebrow}</p>
          <SheetTitle className="text-lg font-bold tracking-[-0.025em]">{section.title}</SheetTitle>
          <SheetDescription>{section.summary}</SheetDescription>
        </SheetHeader>

        <div className="px-4 py-5">
          <GuideBody section={section} />
        </div>

        <div className="mt-auto border-t px-4 py-4">
          <Button asChild variant="outline" className="group/nudge w-full" onClick={() => setOpen(false)}>
            <Link href={`/panduan#${section.id}`}>
              {tg.helpAll} <ArrowUpRight className="nudge size-4" />
            </Link>
          </Button>
        </div>
      </SheetContent>
    </Sheet>
  );
}
