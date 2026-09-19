import { Info, TriangleAlert } from "lucide-react";
import { cn } from "@/lib/utils";
import type { GuideBlock, GuideSection } from "@/features/guide/content";

/** Renders one guide block. Kept dumb so the page and the help sheet share it. */
function Block({ block }: { block: GuideBlock }) {
  switch (block.kind) {
    case "p":
      return <p className="text-sm leading-relaxed text-muted-foreground">{block.text}</p>;

    case "list":
      return (
        <ul className="space-y-1.5">
          {block.items.map((it, i) => (
            <li key={i} className="flex gap-2.5 text-sm leading-relaxed text-muted-foreground">
              <span aria-hidden className="mt-[0.55rem] size-1 shrink-0 rounded-full bg-brand" />
              <span>{it}</span>
            </li>
          ))}
        </ul>
      );

    case "steps":
      return (
        <ol className="space-y-3">
          {block.items.map((it, i) => (
            <li key={i} className="flex gap-3.5">
              <span className="mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full bg-brand text-xs font-semibold text-primary-foreground">
                {i + 1}
              </span>
              <span className="min-w-0">
                <span className="block text-sm font-semibold">{it.title}</span>
                <span className="mt-0.5 block text-sm leading-relaxed text-muted-foreground">{it.text}</span>
              </span>
            </li>
          ))}
        </ol>
      );

    case "defs":
      return (
        <dl className="overflow-hidden rounded-xl border">
          {block.items.map((it, i) => (
            <div key={i} className={cn("px-4 py-3", i > 0 && "border-t")}>
              <dt className="flex flex-wrap items-baseline gap-2">
                <span className="text-sm font-semibold">{it.term}</span>
                {it.formula ? (
                  <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs text-foreground/80">
                    {it.formula}
                  </code>
                ) : null}
              </dt>
              <dd className="mt-1 text-sm leading-relaxed text-muted-foreground">{it.text}</dd>
            </div>
          ))}
        </dl>
      );

    case "note": {
      const warn = block.tone === "warn";
      const Icon = warn ? TriangleAlert : Info;
      return (
        <div
          className="flex gap-2.5 rounded-xl border px-4 py-3"
          style={{
            borderColor: `color-mix(in oklab, ${warn ? "var(--status-warning)" : "var(--brand)"} 35%, transparent)`,
            background: `color-mix(in oklab, ${warn ? "var(--status-warning)" : "var(--brand)"} 7%, transparent)`,
          }}
        >
          <Icon
            className="mt-0.5 size-4 shrink-0"
            style={{ color: warn ? "var(--status-warning)" : "var(--brand)" }}
          />
          <p className="text-sm leading-relaxed text-foreground/85">{block.text}</p>
        </div>
      );
    }
  }
}

export function GuideBody({ section, className }: { section: GuideSection; className?: string }) {
  return (
    <div className={cn("space-y-4", className)}>
      {section.blocks.map((b, i) => (
        <Block key={i} block={b} />
      ))}
    </div>
  );
}
