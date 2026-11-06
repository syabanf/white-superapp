import { notFound, redirect } from "next/navigation";
import { db } from "@/lib/db";

/** Invoice emails link by client id (the services do not know the slug); send the visitor to the share page. */
export default async function ShareByClientId(props: PageProps<"/share/c/[clientId]">) {
  const { clientId } = await props.params;
  const client = await db.client.findUnique({ where: { id: clientId }, select: { slug: true } });
  if (!client) notFound();
  redirect(`/share/${client.slug}`);
}
