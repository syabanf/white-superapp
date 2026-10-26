import type { GatewayError } from "@/lib/erp/gateway";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { ErpError } from "@/features/erp/shared/states";
import { sp } from "../strings";

/** Offline banner for network failures, a permission hint or an inline error otherwise. */
export function SectionError({ error, compact }: { error: GatewayError; compact?: boolean }) {
  if (error.code === "NETWORK") return <OfflineNotice message={error.message} />;
  const denied = error.status === 403 || error.code === "UNAUTHORIZED" || error.code === "PERMISSION_DENIED";
  return <ErpError compact={compact} message={denied ? sp.common.noAccess : error.message} />;
}
