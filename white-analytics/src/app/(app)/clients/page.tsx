import type { Metadata } from "next";
import Link from "next/link";
import { CircleAlert, Plus } from "lucide-react";
import { Alert, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/dashboard/page-header";
import { ClientsTable } from "@/features/clients/components/clients-table";
import { listClientsWithMeta } from "@/features/clients/queries";
import { requireUser } from "@/lib/rbac";
import { t } from "@/i18n/id";
import { tc } from "@/features/clients/strings";

export const metadata: Metadata = { title: t.clients.title };

export default async function ClientsPage(props: PageProps<"/clients">) {
  const sp = await props.searchParams;
  const [user, clients] = await Promise.all([requireUser(), listClientsWithMeta()]);
  const isAdmin = user.role === "ADMIN";
  const showMissing = sp.missing === "1";

  return (
    <>
      <PageHeader
        eyebrow={t.nav.clients}
        title={t.clients.title}
        description={t.clients.subtitle}
        actions={
          isAdmin ? (
            <Button asChild size="sm">
              <Link href="/clients/new">
                <Plus className="size-4" /> {t.clients.new}
              </Link>
            </Button>
          ) : undefined
        }
      />

      {showMissing ? (
        <Alert variant="destructive" className="py-2">
          <CircleAlert className="size-4" />
          <AlertTitle className="text-xs font-medium">{tc.clients.missing}</AlertTitle>
        </Alert>
      ) : null}

      <ClientsTable clients={clients} isAdmin={isAdmin} />
    </>
  );
}
