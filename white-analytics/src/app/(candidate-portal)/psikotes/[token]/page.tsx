import type { Metadata } from "next";
import { loadPublic } from "@/features/erp/recruitment/public/load";
import { PsikotesPortal } from "@/features/erp/recruitment/public/psikotes-portal";
import { PortalProblem } from "@/features/erp/recruitment/public/states";
import { cp } from "@/features/erp/recruitment/strings";
import type { PsikotesPortal as Portal } from "@/features/erp/recruitment/types";

export const metadata: Metadata = { title: cp.psikotes.eyebrow };

export default async function PublicPsikotesPage(props: PageProps<"/psikotes/[token]">) {
  const { token } = await props.params;
  const res = await loadPublic<Portal>(`/v1/recruitment/public/psikotes/${encodeURIComponent(token)}`);
  if (!res.data) return <PortalProblem kind={res.error} message={res.message} />;
  return <PsikotesPortal token={token} initial={res.data} />;
}
