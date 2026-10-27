import type { Metadata } from "next";
import { LoginForm, type DemoAccount } from "@/features/auth/login-form";
import { MixistLogo } from "@/components/brand/logo";
import { MarketingFunnel3D } from "@/components/brand/marketing-funnel-3d";
import { t } from "@/i18n/id";

export const metadata: Metadata = { title: t.common.login };

const HIGHLIGHTS = [
  { name: "Social Media", meta: "Analitik · Publikasi" },
  { name: "SEO", meta: "GSC · Rank · Backlink" },
  { name: "Meta Ads", meta: "Spend · CPR · ROAS" },
];

// Seeded demo accounts. Prod hides them unless NEXT_PUBLIC_DEMO_LOGIN=true, so the
// credentials only reach the browser when the quick-login buttons are meant to show.
const DEMO_LOGIN = process.env.NODE_ENV !== "production" || process.env.NEXT_PUBLIC_DEMO_LOGIN === "true";
const DEMO_ACCOUNTS: DemoAccount[] = [
  { label: "Admin", email: "admin@mixist.id", password: "admin12345" },
  { label: "Tim", email: "tim@mixist.id", password: "member12345" },
];

export default async function LoginPage(props: PageProps<"/login">) {
  const sp = await props.searchParams;
  const callbackUrl = typeof sp.callbackUrl === "string" ? sp.callbackUrl : undefined;

  return (
    <main className="relative min-h-dvh bg-background">
      <div className="mx-auto grid min-h-dvh w-full max-w-6xl items-center gap-10 px-6 py-14 lg:grid-cols-[1.05fr_minmax(380px,420px)] lg:gap-16">
        {/* ── Hero ── */}
        <section className="hidden lg:block">
          <MixistLogo size={40} />
          <h1 className="mt-10 text-[52px] leading-[1.05] font-semibold tracking-[-0.03em]">
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
                <li key={h.name} className="rounded-2xl bg-card px-3.5 py-3 shadow-(--card-shadow)">
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
          <div className="rounded-[2rem] bg-card p-7 shadow-(--card-shadow) sm:p-9">
            <div className="lg:hidden">
              <MixistLogo size={34} />
            </div>
            <p className="label-mono mt-6 text-muted-foreground lg:mt-0">Masuk</p>
            <h2 className="mt-1.5 text-[26px] leading-tight font-semibold tracking-[-0.03em]">
              {t.auth.title}
            </h2>
            <p className="mt-2 mb-7 text-sm text-muted-foreground">{t.auth.subtitle}</p>

            <LoginForm callbackUrl={callbackUrl} demoAccounts={DEMO_LOGIN ? DEMO_ACCOUNTS : undefined} />
          </div>
          <p className="label-mono mt-6 text-center text-muted-foreground">
            © {new Date().getFullYear()} {t.app.name}
          </p>
        </section>
      </div>
    </main>
  );
}
