import type { Metadata } from "next";
import { loadPublic } from "@/features/erp/sales/public/load";
import { OnboardingPortal } from "@/features/erp/sales/public/onboarding-portal";
import { PublicProblem } from "@/features/erp/sales/public/states";
import { ps } from "@/features/erp/sales/strings";
import type { PublicOnboarding } from "@/features/erp/sales/types";

export const metadata: Metadata = { title: ps.onboarding.eyebrow };

export default async function PublicOnboardingPage(props: PageProps<"/onboarding/[token]">) {
  const { token } = await props.params;
  const res = await loadPublic<PublicOnboarding>(`/v1/public/sales/onboarding/${encodeURIComponent(token)}`);
  if (!res.data) return <PublicProblem kind={res.error} message={res.message} />;
  return (
    <>
      <div>
        <p className="label-mono text-muted-foreground">{ps.onboarding.eyebrow}</p>
        <h1 className="mt-1 text-[26px] leading-tight font-semibold">{ps.onboarding.welcome(res.data.clientName)}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{ps.onboarding.intro}</p>
      </div>
      <OnboardingPortal token={token} initial={res.data} />
    </>
  );
}
