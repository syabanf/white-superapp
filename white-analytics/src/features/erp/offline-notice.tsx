import { ServerCrash } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { t } from "@/i18n/id";

/** Shown when the Go gateway is unreachable, so ERP pages fail loudly and clearly. */
export function OfflineNotice({ message }: { message: string }) {
  return (
    <Alert variant="destructive" className="rounded-2xl">
      <ServerCrash className="size-4" />
      <AlertTitle>{t.erp.offlineTitle}</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  );
}
