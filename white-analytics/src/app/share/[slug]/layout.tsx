import { Sparkles } from "lucide-react";
import { MixistLogo } from "@/components/brand/logo";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { ShareControls } from "@/features/share/components/share-controls";
import { getShareContext } from "@/features/share/access";
import { t } from "@/i18n/id";

/** Minimal public chrome: top bar (logo · client · live badge · range/PDF controls) + footer. */
export default async function ShareLayout(props: LayoutProps<"/share/[slug]">) {
  const { slug } = await props.params;
  const ctx = await getShareContext(slug);
  const client = ctx.status === "not-found" ? null : ctx.client;

  return (
    <div className="flex min-h-svh flex-1 flex-col bg-background [--page-max:72rem]">
      <header className="sticky top-0 z-20 border-b bg-background">
        <div className="page-container flex h-14 items-center gap-3">
          <MixistLogo size={28} />
          {client && ctx.status !== "disabled" ? (
            <>
              <Separator orientation="vertical" className="!h-5" />
              <span className="min-w-0 truncate text-sm font-medium" title={client.name}>
                {client.name}
              </span>
              <Badge
                variant="outline"
                className="hidden gap-1 border-dashed text-xs text-muted-foreground sm:inline-flex"
              >
                <Sparkles className="size-3" /> {t.share.live}
              </Badge>
            </>
          ) : null}
          <div className="ml-auto shrink-0">{ctx.status === "ok" ? <ShareControls slug={slug} /> : null}</div>
        </div>
      </header>
      <main className="page-enter page-container page-stack flex-1 py-6 md:py-8">
        {props.children}
      </main>
      <footer className="border-t">
        <div className="page-container flex items-center justify-between gap-3 py-4 text-xs text-muted-foreground">
          <span>{t.share.footer}</span>
          <MixistLogo variant="mark" size={16} className="text-muted-foreground" />
        </div>
      </footer>
    </div>
  );
}
