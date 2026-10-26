import { CloudOff } from "lucide-react";
import { EmptyState } from "@/components/dashboard/empty-state";
import { c } from "@/features/content/strings";

export function ContentOffline({ message }: { message?: string }) {
  return <EmptyState state="disconnected" icon={<CloudOff />} title={c.offlineTitle} description={message ?? c.offlineDesc} />;
}
