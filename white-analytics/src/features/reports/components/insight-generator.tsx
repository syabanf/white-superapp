"use client";

import * as React from "react";
import { useTransition } from "react";
import { useRouter } from "next/navigation";
import { Loader2, Sparkles } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { generateInsight } from "@/features/reports/actions";
import type { InsightModule } from "@/features/reports/queries";
import { t } from "@/i18n/id";
import { rs } from "@/features/reports/strings";

/**
 * Small generate/regenerate button for a module insight (used on Overview & Reports).
 * Props are plain strings so any RSC can render it.
 */
export function GenerateInsightButton({
  slug,
  module,
  from,
  to,
  language = "id",
  hasExisting = false,
  size = "sm",
  variant = "outline",
  className,
}: {
  slug: string;
  module: InsightModule;
  from: string;
  to: string;
  language?: "id" | "en";
  hasExisting?: boolean;
  size?: "sm" | "default";
  variant?: "outline" | "default" | "ghost";
  className?: string;
}) {
  const router = useRouter();
  const [pending, startTransition] = useTransition();

  const run = () => {
    startTransition(async () => {
      const res = await generateInsight(slug, module, from, to, language);
      if (res.ok) {
        toast.success(rs.insight.generated);
        router.refresh();
      } else {
        toast.error(res.error);
      }
    });
  };

  return (
    <Button type="button" size={size} variant={variant} onClick={run} disabled={pending} className={className}>
      {pending ? <Loader2 className="size-3.5 animate-spin" /> : <Sparkles className="size-3.5" />}
      {pending ? t.common.generating : hasExisting ? t.reports.aiRegenerate : t.reports.aiGenerate}
    </Button>
  );
}

/** Generate button + id/en language toggle — the action row on each insight card. */
export function InsightGenerator({
  slug,
  module,
  from,
  to,
  hasExisting,
}: {
  slug: string;
  module: InsightModule;
  from: string;
  to: string;
  hasExisting: boolean;
}) {
  const [language, setLanguage] = React.useState<"id" | "en">("id");
  return (
    <div className="flex items-center gap-2">
      <ToggleGroup
        type="single"
        size="sm"
        variant="outline"
        value={language}
        onValueChange={(v) => v && setLanguage(v as "id" | "en")}
        aria-label={t.common.language}
      >
        <ToggleGroupItem value="id" className="h-7 px-2 text-xs">
          ID
        </ToggleGroupItem>
        <ToggleGroupItem value="en" className="h-7 px-2 text-xs">
          EN
        </ToggleGroupItem>
      </ToggleGroup>
      <GenerateInsightButton slug={slug} module={module} from={from} to={to} language={language} hasExisting={hasExisting} />
    </div>
  );
}
