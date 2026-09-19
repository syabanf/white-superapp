import { Sparkles } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { t } from "@/i18n/id";

/**
 * AI strategic summary card. Rendered by the Overview page; the Reports/AI module
 * provides the generation action + cached insight lookup (see features/reports).
 * This is the shared shell so the page contract is stable.
 */
export function AiInsightCard({
  content,
  createdAt,
  action,
  isMock,
}: {
  content?: string | null;
  createdAt?: Date | null;
  action?: React.ReactNode;
  isMock?: boolean;
}) {
  return (
    <Card className="gap-3 py-0">
      <CardHeader className="flex flex-row items-start justify-between gap-3 space-y-0 px-6 pt-5 pb-0">
        <div>
          <CardTitle className="flex items-center gap-2 text-sm font-semibold">
            <Sparkles className="size-4 text-muted-foreground" /> {t.reports.aiInsight}
          </CardTitle>
          <CardDescription className="mt-0.5 text-xs">{t.reports.aiInsightDesc}</CardDescription>
        </div>
        {content ? action : null}
      </CardHeader>
      <CardContent className="px-6 pb-6">
        {content ? (
          <div className="prose prose-sm max-w-none text-sm text-foreground/90 dark:prose-invert [&_h3]:mt-3 [&_h3]:text-sm [&_ol]:pl-4 [&_ul]:pl-4">
            <MarkdownLite text={content} />
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center gap-3 py-10 text-center">
            <span className="flex size-11 items-center justify-center rounded-full bg-brand/8 text-brand ring-1 ring-brand/20">
              <Sparkles className="size-5" />
            </span>
            <div>
              <p className="text-sm font-medium">{t.reports.aiEmpty}</p>
              <p className="mt-1 max-w-sm text-xs text-muted-foreground">{t.reports.aiInsightDesc}</p>
            </div>
            {action ? <div className="mt-1">{action}</div> : null}
          </div>
        )}
        {createdAt || isMock ? (
          <p className="mt-3 text-xs text-muted-foreground">
            {isMock ? t.reports.aiMock : null}
            {createdAt
              ? ` ${t.reports.aiCached} ${createdAt.toLocaleString("id-ID", { timeZone: "Asia/Jakarta" })}`
              : null}
          </p>
        ) : null}
      </CardContent>
    </Card>
  );
}

/** Minimal markdown: **bold**, headings (#, ##, ###), bullets (-, *), numbered lists. No raw HTML. */
export function MarkdownLite({ text }: { text: string }) {
  const lines = text.split(/\r?\n/);
  const out: React.ReactNode[] = [];
  // Keys are prefixed per block type ("p-3" vs "ul-3") so list keys never collide with line keys.
  let list: { type: "ul" | "ol"; startLine: number; items: React.ReactNode[] } | null = null;
  const flush = () => {
    if (list) {
      const key = `${list.type}-${list.startLine}`;
      out.push(
        list.type === "ul" ? (
          <ul key={key} className="list-disc space-y-0.5">
            {list.items}
          </ul>
        ) : (
          <ol key={key} className="list-decimal space-y-0.5">
            {list.items}
          </ol>
        ),
      );
      list = null;
    }
  };
  lines.forEach((raw, i) => {
    const line = raw.trimEnd();
    if (!line.trim()) {
      flush();
      return;
    }
    const h = /^(#{1,3})\s+(.*)$/.exec(line);
    if (h) {
      flush();
      out.push(
        <h3 key={`h-${i}`} className="font-semibold">
          {inline(h[2]!)}
        </h3>,
      );
      return;
    }
    const ul = /^\s*[-*•]\s+(.*)$/.exec(line);
    if (ul) {
      if (!list || list.type !== "ul") {
        flush();
        list = { type: "ul", startLine: i, items: [] };
      }
      list.items.push(<li key={i}>{inline(ul[1]!)}</li>);
      return;
    }
    const ol = /^\s*\d+[.)]\s+(.*)$/.exec(line);
    if (ol) {
      if (!list || list.type !== "ol") {
        flush();
        list = { type: "ol", startLine: i, items: [] };
      }
      list.items.push(<li key={i}>{inline(ol[1]!)}</li>);
      return;
    }
    flush();
    out.push(<p key={`p-${i}`}>{inline(line)}</p>);
  });
  flush();
  return <>{out}</>;
}

function inline(s: string): React.ReactNode {
  const parts = s.split(/(\*\*[^*]+\*\*)/g);
  return parts.map((p, i) =>
    p.startsWith("**") && p.endsWith("**") ? (
      <strong key={i}>{p.slice(2, -2)}</strong>
    ) : (
      <span key={i}>{p}</span>
    ),
  );
}
