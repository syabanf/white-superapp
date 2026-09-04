import { requireClientAccess } from "@/lib/rbac";

/** Verifies access once per request for every client page (RBAC in the data layer, not only UI). */
export default async function ClientLayout({ children, params }: LayoutProps<"/clients/[slug]">) {
  const { slug } = await params;
  await requireClientAccess(slug);
  return <>{children}</>;
}
