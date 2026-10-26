import type { Metadata } from "next";
import { InterviewPortal } from "@/features/erp/recruitment/public/interview-portal";
import { loadPublic } from "@/features/erp/recruitment/public/load";
import { PortalProblem } from "@/features/erp/recruitment/public/states";
import { cp } from "@/features/erp/recruitment/strings";
import type { InterviewPortal as Portal } from "@/features/erp/recruitment/types";

export const metadata: Metadata = { title: cp.interview.eyebrow };

export default async function PublicInterviewPage(props: PageProps<"/wawancara/[token]">) {
  const { token } = await props.params;
  const res = await loadPublic<Portal>(`/v1/recruitment/public/interviews/${encodeURIComponent(token)}`);
  if (!res.data) return <PortalProblem kind={res.error} message={res.message} />;
  return <InterviewPortal token={token} initial={res.data} />;
}
