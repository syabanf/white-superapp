import { CloudOff, SearchX } from "lucide-react";
import { EmptyState } from "@/components/dashboard/empty-state";
import { ps } from "../strings";

/** Not-found or outage message for a public page. */
export function PublicProblem({ kind, message }: { kind: "not-found" | "offline" | "error"; message: string }) {
  if (kind === "not-found")
    return <EmptyState icon={<SearchX className="size-5" />} title={ps.notFound} description={ps.notFoundDesc} className="my-auto" />;
  return <EmptyState state="error" icon={<CloudOff className="size-5" />} title={ps.offline} description={message} className="my-auto" />;
}
