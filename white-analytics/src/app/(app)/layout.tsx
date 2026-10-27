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
      <SidebarProvider defaultOpen={defaultOpen} className="app-frame">
        <AppSidebar clients={switcherClients} user={shellUser} erpAccess={erpAccess} />
        <SidebarInset className="min-w-0 bg-background md:my-3 md:mr-3 md:min-h-[calc(100svh-1.5rem)] md:rounded-[2rem] md:surface-panel">
          <Topbar clients={switcherClients} notifications={notifications} />
          {/* The page container: every page shares its edges, gap and top spacing (LAYOUT.md).
              pb-28 on phones clears the floating tab bar. */}
          <div className="page-enter page-container page-stack flex-1 pt-2 pb-28 md:pb-8 lg:pb-10">
            {children}
          </div>
          <MobileTabBar clients={switcherClients} user={shellUser} erpAccess={erpAccess} />
        </SidebarInset>
      </SidebarProvider>
    </WorkspaceScope>
  );
}
