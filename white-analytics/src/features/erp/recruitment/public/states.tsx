import type { ReactNode } from "react";
import { CloudOff, SearchX } from "lucide-react";
import { EmptyState } from "@/components/dashboard/empty-state";
import { cp } from "../strings";

/** Not-found or outage message for a public page. */
export function PortalProblem({ kind, message }: { kind: "not-found" | "offline" | "error"; message: string }) {
  if (kind === "not-found")
    return <EmptyState icon={<SearchX className="size-5" />} title={cp.notFound} description={cp.notFoundDesc} className="my-auto" />;
  return <EmptyState state="error" icon={<CloudOff className="size-5" />} title={kind === "offline" ? cp.offline : cp.error} description={message} className="my-auto" />;
}

/** Full-width end state (finished, expired). */
export function PortalNotice({ icon, title, description }: { icon: ReactNode; title: string; description: string }) {
  return <EmptyState icon={icon} title={title} description={description} className="my-auto" />;
}
