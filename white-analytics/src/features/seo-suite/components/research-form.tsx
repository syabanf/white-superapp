import Link from "next/link";
import { Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import type { ResearchMode } from "@/lib/providers/dataforseo/types";
import { MODE_LABELS, RESEARCH_MODES } from "@/features/seo-suite/lib";
import { s } from "@/features/seo-suite/strings";

/** GET form (works without JS): ?seed=…&mode=… — results are cached server-side for 7 days. */
export function ResearchForm({ seed, mode, basePath, gap }: { seed: string; mode: ResearchMode; basePath: string; gap?: string }) {
  const hrefFor = (m: ResearchMode) => {
    const q = new URLSearchParams();
    if (seed) q.set("seed", seed);
    if (m !== "ideas") q.set("mode", m);
    if (gap) q.set("gap", gap);
    const qs = q.toString();
    return qs ? `${basePath}?${qs}` : basePath;
  };
  return (
    <div className="flex flex-col gap-3">
      <form method="get" action={basePath} className="flex flex-col gap-2 sm:flex-row sm:items-end">
        <input type="hidden" name="mode" value={mode} />
        {gap ? <input type="hidden" name="gap" value={gap} /> : null}
        <div className="min-w-0 flex-1 space-y-1.5">
          <Label htmlFor="seed" className="label-mono">
            {s.seedLabel}
          </Label>
          <Input id="seed" name="seed" defaultValue={seed} placeholder={s.seedPlaceholder} autoComplete="off" required maxLength={120} />
        </div>
        <Button type="submit" className="sm:self-end">
          <Search className="size-4" /> {s.searchIdeas}
        </Button>
      </form>
      <div className="flex w-fit max-w-full flex-wrap items-center gap-1 rounded-lg bg-muted p-1">
        {RESEARCH_MODES.map((m) => {
          const active = m === mode;
          return (
            <Link
              key={m}
              href={hrefFor(m)}
              aria-current={active ? "page" : undefined}
              className={cn(
                "inline-flex items-center rounded-md px-2.5 py-1 text-xs font-medium transition-colors",
                active ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground",
              )}
            >
              {MODE_LABELS[m]}
            </Link>
          );
        })}
      </div>
    </div>
  );
}
