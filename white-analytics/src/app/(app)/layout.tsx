import { cookies } from "next/headers";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/shell/app-sidebar";
import { Topbar } from "@/components/shell/topbar";
import { MobileTabBar } from "@/components/shell/mobile-tab-bar";
import { listAccessibleClients, requireUser } from "@/lib/rbac";
import { getNotificationFeed } from "@/features/notifications/queries";

export default async function AppLayout({ children }: LayoutProps<"/">) {
  const user = await requireUser();
  const [clients, cookieStore, notifications] = await Promise.all([listAccessibleClients(), cookies(), getNotificationFeed(user.id)]);
  const defaultOpen = cookieStore.get("sidebar_state")?.value !== "false";
  const switcherClients = clients.map((c) => ({ id: c.id, name: c.name, slug: c.slug, industry: c.industry }));

  const shellUser = { id: user.id, name: user.name, email: user.email, role: user.role };

  return (
    <SidebarProvider defaultOpen={defaultOpen}>
      <AppSidebar clients={switcherClients} user={shellUser} />
      <SidebarInset className="min-w-0">
        <Topbar clients={switcherClients} notifications={notifications} />
        {/* pb-28 on phones clears the floating tab bar */}
        <div className="page-enter mx-auto flex w-full max-w-[1600px] flex-1 flex-col gap-6 px-4 pt-2 pb-28 md:gap-7 md:px-6 md:pb-8 lg:px-8 lg:pb-10">{children}</div>
        <MobileTabBar clients={switcherClients} user={shellUser} />
      </SidebarInset>
    </SidebarProvider>
  );
}
