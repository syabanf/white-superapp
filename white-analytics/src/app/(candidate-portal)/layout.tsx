import type { Metadata } from "next";
import { MixistLogo } from "@/components/brand/logo";
import { cp } from "@/features/erp/recruitment/strings";

export const metadata: Metadata = { robots: { index: false, follow: false } };

/** Minimal public chrome for candidate pages (no login). */
export default function CandidatePortalLayout({ children }: LayoutProps<"/">) {
  return (
    <div className="flex min-h-svh flex-1 flex-col bg-background [--page-max:48rem]">
      <header className="border-b bg-background">
        <div className="page-container flex h-14 items-center">
          <MixistLogo size={28} />
        </div>
      </header>
      <main className="page-enter page-container page-stack flex-1 py-6 md:py-10">{children}</main>
      <footer className="border-t">
        <div className="page-container flex items-center justify-between gap-3 py-4 text-xs text-muted-foreground">
          <span>{cp.footer}</span>
          <MixistLogo variant="mark" size={16} className="text-muted-foreground" />
        </div>
      </footer>
    </div>
  );
}
