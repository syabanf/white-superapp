import { Sparkles } from "lucide-react";
import { WhiteLogo } from "@/components/brand/logo";
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
    <div className="flex min-h-svh flex-1 flex-col bg-canvas">
      <header className="sticky top-0 z-20 border-b bg-canvas/90 backdrop-blur">
        <div className="mx-auto flex h-14 w-full max-w-6xl items-center gap-3 px-4 md:px-6">
          <WhiteLogo size={20} />
          {client && ctx.status !== "disabled" ? (
            <>
              <Separator orientation="vertical" className="!h-5" />
              <span className="truncate text-sm font-medium">{client.name}</span>
              <Badge variant="outline" className="hidden gap-1 border-dashed text-[11px] text-muted-foreground sm:inline-flex">
                <Sparkles className="size-3" /> {t.share.live}
              </Badge>
            </>
          ) : null}
          <div className="ml-auto">{ctx.status === "ok" ? <ShareControls slug={slug} /> : null}</div>
        </div>
      </header>
      <main className="page-enter mx-auto flex w-full max-w-6xl flex-1 flex-col gap-6 px-4 py-6 md:px-6 md:py-8">{props.children}</main>
      <footer className="border-t">
        <div className="mx-auto flex w-full max-w-6xl items-center justify-between gap-3 px-4 py-4 text-xs text-muted-foreground md:px-6">
          <span>{t.share.footer}</span>
          <WhiteLogo variant="mark" size={14} className="text-muted-foreground" />
        </div>
      </footer>
    </div>
  );
}
