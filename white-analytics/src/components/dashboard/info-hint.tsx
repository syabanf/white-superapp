import type { ReactNode } from "react";
import { Info } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

/**
 * The one place explanatory copy lives: an info icon that opens a tooltip.
 * Titles stay short on screen; the "what is this" sentence goes here.
 */
export function InfoHint({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={typeof children === "string" ? children : undefined}
          className={cn("shrink-0 text-muted-foreground/50 transition-colors hover:text-muted-foreground", className)}
        >
          <Info className="size-3.5" />
        </button>
      </TooltipTrigger>
      <TooltipContent className="max-w-64 text-xs leading-relaxed">{children}</TooltipContent>
    </Tooltip>
  );
}
