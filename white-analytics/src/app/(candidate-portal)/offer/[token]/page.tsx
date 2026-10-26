import type { Metadata } from "next";
import { loadPublic } from "@/features/erp/recruitment/public/load";
import { OfferPortal } from "@/features/erp/recruitment/public/offer-portal";
import { PortalProblem } from "@/features/erp/recruitment/public/states";
import { cp } from "@/features/erp/recruitment/strings";
import type { OfferPortal as Portal } from "@/features/erp/recruitment/types";

export const metadata: Metadata = { title: cp.offer.eyebrow };

export default async function PublicOfferPage(props: PageProps<"/offer/[token]">) {
  const { token } = await props.params;
  const res = await loadPublic<Portal>(`/v1/recruitment/public/offers/${encodeURIComponent(token)}`);
  if (!res.data) return <PortalProblem kind={res.error} message={res.message} />;
  return <OfferPortal token={token} initial={res.data} />;
}
