import type { Metadata } from "next";
import { WhiteLogo } from "@/components/brand/logo";
import { cp } from "@/features/erp/recruitment/strings";

export const metadata: Metadata = { robots: { index: false, follow: false } };

/** Minimal public chrome for candidate pages (no login). */
export default function CandidatePortalLayout({ children }: LayoutProps<"/">) {
  return (
    <div className="flex min-h-svh flex-1 flex-col bg-canvas">
      <header className="border-b bg-canvas/90 backdrop-blur">
        <div className="mx-auto flex h-14 w-full max-w-3xl items-center px-4 md:px-6">
          <WhiteLogo size={20} />
        </div>
      </header>
      <main className="page-enter mx-auto flex w-full max-w-3xl flex-1 flex-col gap-6 px-4 py-6 md:px-6 md:py-10">{children}</main>
      <footer className="border-t">
        <div className="mx-auto flex w-full max-w-3xl items-center justify-between gap-3 px-4 py-4 text-xs text-muted-foreground md:px-6">
          <span>{cp.footer}</span>
          <WhiteLogo variant="mark" size={14} className="text-muted-foreground" />
        </div>
      </footer>
    </div>
  );
}
