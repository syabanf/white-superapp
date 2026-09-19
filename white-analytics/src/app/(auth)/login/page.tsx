import type { Metadata } from "next";
import { LoginForm } from "@/features/auth/login-form";
import { WhiteLogo } from "@/components/brand/logo";
import { MarketingFunnel3D } from "@/components/brand/marketing-funnel-3d";
import { t } from "@/i18n/id";

export const metadata: Metadata = { title: t.common.login };

const HIGHLIGHTS = [
  { name: "Social Media", meta: "Analitik · Publikasi" },
  { name: "SEO", meta: "GSC · Rank · Backlink" },
  { name: "Meta Ads", meta: "Spend · CPR · ROAS" },
];

export default async function LoginPage(props: PageProps<"/login">) {
  const sp = await props.searchParams;
  const callbackUrl = typeof sp.callbackUrl === "string" ? sp.callbackUrl : undefined;

  return (
    <main className="relative min-h-dvh bg-canvas">
      <div className="mx-auto grid min-h-dvh w-full max-w-6xl items-center gap-10 px-6 py-14 lg:grid-cols-[1.05fr_minmax(380px,420px)] lg:gap-16">
        {/* ── Hero ── */}
        <section className="hidden lg:block">
          <WhiteLogo size={30} />
          <h1 className="mt-10 text-[52px] leading-[1.02] font-medium tracking-[-0.035em]">
            Tiga kanal, <span className="text-muted-foreground">satu corong.</span>
          </h1>
          <p className="mt-5 max-w-md text-[15px] leading-relaxed text-muted-foreground">
            Pantau Social Media, SEO, dan Meta Ads setiap klien, jadwalkan kontennya, lalu kirim laporannya
            dalam satu tautan.
          </p>

          <div className="stage-tint relative mt-8 h-[340px] max-w-xl overflow-hidden rounded-[1.75rem]">
            <div className="absolute inset-x-0 top-0 h-[270px]">
              <MarketingFunnel3D />
            </div>
            <ul className="absolute inset-x-4 bottom-4 grid grid-cols-3 gap-2.5">
              {HIGHLIGHTS.map((h) => (
                <li key={h.name} className="surface-glass rounded-2xl px-3.5 py-3">
                  <span className="block text-sm font-semibold tracking-[-0.02em]">{h.name}</span>
                  <span className="mt-0.5 block text-xs text-muted-foreground">{h.meta}</span>
                </li>
              ))}
            </ul>
          </div>
        </section>

        {/* ── Form ── */}
        <section className="mx-auto w-full max-w-[420px]">
          <div className="stage-tint mb-4 h-44 overflow-hidden rounded-[1.5rem] lg:hidden">
            <MarketingFunnel3D />
          </div>
          <div className="rounded-[1.75rem] bg-card p-7 shadow-(--card-shadow) sm:p-9">
            <div className="lg:hidden">
              <WhiteLogo size={26} />
            </div>
            <p className="label-mono mt-6 text-muted-foreground lg:mt-0">Masuk</p>
            <h2 className="mt-1.5 text-[26px] leading-tight font-semibold tracking-[-0.03em]">
              {t.auth.title}
            </h2>
            <p className="mt-2 mb-7 text-sm text-muted-foreground">{t.auth.subtitle}</p>

            <LoginForm callbackUrl={callbackUrl} />

            <div className="mt-7 border-t pt-5">
              <p className="label-mono text-muted-foreground">{t.auth.demoHint}</p>
              <dl className="mt-2.5 space-y-1 font-mono text-xs text-muted-foreground">
                <div className="flex justify-between gap-3">
                  <dt className="text-foreground/80">admin@white.id</dt>
                  <dd>admin12345</dd>
                </div>
                <div className="flex justify-between gap-3">
                  <dt className="text-foreground/80">tim@white.id</dt>
                  <dd>member12345</dd>
                </div>
              </dl>
            </div>
          </div>
          <p className="label-mono mt-6 text-center text-muted-foreground">
            © {new Date().getFullYear()} WHITE · {t.app.name}
          </p>
        </section>
      </div>
    </main>
  );
}
