import type { Metadata } from "next";
import { LoginForm } from "@/features/auth/login-form";
import { WhiteLogo } from "@/components/brand/logo";
import { t } from "@/i18n/id";

export const metadata: Metadata = { title: t.common.login };

const HIGHLIGHTS = [
  { index: "01", name: "Social Media", meta: "IG · FB · TIKTOK" },
  { index: "02", name: "SEO", meta: "GSC · GA4 · AUDIT" },
  { index: "03", name: "Meta Ads", meta: "SPEND · CPR · ROAS" },
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
          <p className="label-mono mt-12 text-brand">{t.app.tagline}</p>
          <h1 className="mt-4 text-[56px] leading-[0.98] font-bold tracking-[-0.04em]">
            Data klien,
            <br />
            satu meja.
          </h1>
          <span aria-hidden className="mt-5 block h-[3px] w-24 rounded-full bg-brand" />
          <p className="mt-6 max-w-md text-[15px] leading-relaxed text-muted-foreground">
            Pantau performa Social Media, SEO, dan Meta Ads setiap klien — lalu kirim laporannya
            dalam satu tautan.
          </p>

          <ul className="mt-14 grid max-w-lg grid-cols-3 gap-3">
            {HIGHLIGHTS.map((h) => (
              <li key={h.index} className="crop-marks rounded-xl border bg-card/70 px-3.5 py-3.5">
                <span className="label-mono text-brand">{h.index}</span>
                <span className="mt-1.5 block text-sm font-semibold tracking-[-0.02em]">{h.name}</span>
                <span className="label-mono mt-1 block text-[9px] text-muted-foreground">{h.meta}</span>
              </li>
            ))}
          </ul>
        </section>

        {/* ── Form ── */}
        <section className="mx-auto w-full max-w-[420px]">
          <div className="crop-marks rounded-2xl border bg-card p-7 shadow-[0_1px_2px_rgba(10,10,10,0.04),0_12px_40px_-16px_rgba(10,10,10,0.16)] sm:p-9">
            <div className="lg:hidden">
              <WhiteLogo size={26} />
            </div>
            <p className="label-mono mt-6 text-brand lg:mt-0">Masuk</p>
            <h2 className="mt-2 text-[24px] leading-tight font-bold tracking-[-0.03em]">
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
