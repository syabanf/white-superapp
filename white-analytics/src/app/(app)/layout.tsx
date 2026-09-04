import { cookies } from "next/headers";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/shell/app-sidebar";
import { Topbar } from "@/components/shell/topbar";
import { listAccessibleClients, requireUser } from "@/lib/rbac";
import { getNotificationFeed } from "@/features/notifications/queries";

export default async function AppLayout({ children }: LayoutProps<"/">) {
  const user = await requireUser();
  const [clients, cookieStore, notifications] = await Promise.all([listAccessibleClients(), cookies(), getNotificationFeed(user.id)]);
  const defaultOpen = cookieStore.get("sidebar_state")?.value !== "false";
  const switcherClients = clients.map((c) => ({ id: c.id, name: c.name, slug: c.slug, industry: c.industry }));

  return (
    <SidebarProvider defaultOpen={defaultOpen}>
      <AppSidebar clients={switcherClients} user={{ id: user.id, name: user.name, email: user.email, role: user.role }} />
      <SidebarInset className="min-w-0">
        <Topbar clients={switcherClients} notifications={notifications} />
        <div className="page-enter flex flex-1 flex-col gap-6 p-4 md:p-6 lg:p-8">{children}</div>
      </SidebarInset>
    </SidebarProvider>
  );
}
