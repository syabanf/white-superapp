import { cookies } from "next/headers";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/shell/app-sidebar";
import { Topbar } from "@/components/shell/topbar";
import { MobileTabBar } from "@/components/shell/mobile-tab-bar";
import { listAccessibleClients, requireUser } from "@/lib/rbac";
import { getNotificationFeed } from "@/features/notifications/queries";
import { WorkspaceScope } from "@/components/shell/workspace-scope";
import { gatewaySession } from "@/lib/erp/gateway";
import type { ErpAccess } from "@/components/shell/nav-config";

/** ERP permissions from the gateway token; the role alone when the backend is down. */
async function loadErpAccess(role: string): Promise<ErpAccess> {
  try {
    const session = await gatewaySession();
    if (session) return { role: session.role, permissions: session.permissions };
  } catch {
    // Backend unreachable: admins still see the ERP menu and get a clear error on the pages.
  }
  return { role, permissions: [] };
}

export default async function AppLayout({ children }: LayoutProps<"/">) {
  const user = await requireUser();
  const [clients, cookieStore, notifications, erpAccess] = await Promise.all([
    listAccessibleClients(),
    cookies(),
    getNotificationFeed(user.id),
    loadErpAccess(user.role),
  ]);
  const defaultOpen = cookieStore.get("sidebar_state")?.value !== "false";
  const switcherClients = clients.map((c) => ({
    id: c.id,
    name: c.name,
    slug: c.slug,
    industry: c.industry,
  }));

  const shellUser = { id: user.id, name: user.name, email: user.email, role: user.role };

  return (
    <WorkspaceScope>
      <SidebarProvider defaultOpen={defaultOpen}>
        <AppSidebar clients={switcherClients} user={shellUser} erpAccess={erpAccess} />
        <SidebarInset className="min-w-0">
          <Topbar clients={switcherClients} notifications={notifications} />
          {/* pb-28 on phones clears the floating tab bar */}
          <div className="page-enter mx-auto flex w-full max-w-[1600px] flex-1 flex-col gap-6 px-4 pt-2 pb-28 md:gap-7 md:px-6 md:pb-8 lg:px-8 lg:pb-10">
            {children}
          </div>
          <MobileTabBar clients={switcherClients} user={shellUser} erpAccess={erpAccess} />
        </SidebarInset>
      </SidebarProvider>
    </WorkspaceScope>
  );
}
