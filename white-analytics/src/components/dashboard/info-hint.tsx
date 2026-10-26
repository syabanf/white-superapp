"use client";

import { useRef, useState, type ReactNode } from "react";
import { Info } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

/**
 * The one place explanatory copy lives: an info icon that opens a tooltip.
 * Titles stay short on screen; the "what is this" sentence goes here.
 * Hover opens it on desktop; a tap toggles it on touch screens, where Radix
 * tooltips would otherwise never open.
 */
export function InfoHint({ children, className }: { children: ReactNode; className?: string }) {
  const [open, setOpen] = useState(false);
  // Radix closes the tooltip on any outside pointerdown, the trigger included,
  // so remember the state from before the tap to let a second tap close it.
  const wasOpen = useRef(false);
  return (
    <Tooltip open={open} onOpenChange={setOpen}>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={typeof children === "string" ? children : undefined}
          className={cn("shrink-0 text-muted-foreground/50 transition-colors hover:text-muted-foreground", className)}
          // preventDefault skips Radix's own "close on press" handlers.
          onPointerDown={(e) => {
            wasOpen.current = open;
            e.preventDefault();
          }}
          onClick={(e) => {
            e.preventDefault();
            setOpen(!wasOpen.current);
          }}
        >
          <Info className="size-3.5" />
        </button>
      </TooltipTrigger>
      <TooltipContent className="max-w-64 text-xs leading-relaxed">{children}</TooltipContent>
    </Tooltip>
  );
}
