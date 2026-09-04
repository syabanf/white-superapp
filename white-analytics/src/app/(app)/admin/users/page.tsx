import type { Metadata } from "next";
import { UserPlus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/dashboard/page-header";
import { UserFormDialog } from "@/features/admin/components/user-form-dialog";
import { UsersTable } from "@/features/admin/components/users-table";
import { listClientOptions, listUsersWithClients } from "@/features/admin/queries";
import { requireAdmin } from "@/lib/rbac";
import { t } from "@/i18n/id";
import { ta } from "@/features/admin/strings";

export const metadata: Metadata = { title: t.admin.title };

export default async function AdminUsersPage() {
  const admin = await requireAdmin();
  const [users, clients] = await Promise.all([listUsersWithClients(), listClientOptions()]);

  return (
    <>
      <PageHeader
        eyebrow={t.nav.admin}
        title={t.admin.title}
        description={`${t.admin.subtitle} · ${users.length} ${ta.usersCount}`}
        actions={
          <UserFormDialog
            mode="create"
            clients={clients}
            trigger={
              <Button size="sm">
                <UserPlus className="size-4" /> {t.admin.newUser}
              </Button>
            }
          />
        }
      />
      <UsersTable users={users} clients={clients} currentUserId={admin.id} />
    </>
  );
}
