import type { Metadata } from "next";
import { Card } from "@/components/ui/card";
import { ChatBubble } from "@/features/erp/sales/public/chat-widget";
import { loadPublic } from "@/features/erp/sales/public/load";
import { PublicLeadForm } from "@/features/erp/sales/public/public-form";
import { PublicProblem } from "@/features/erp/sales/public/states";
import type { PublicForm } from "@/features/erp/sales/types";

const path = (slug: string) => `/v1/public/sales/forms/${encodeURIComponent(slug)}`;

export async function generateMetadata(props: PageProps<"/f/[slug]">): Promise<Metadata> {
  const { slug } = await props.params;
  const res = await loadPublic<PublicForm>(path(slug));
  return { title: res.data?.name ?? "Formulir", description: res.data?.description };
}

export default async function PublicFormPage(props: PageProps<"/f/[slug]">) {
  const { slug } = await props.params;
  const res = await loadPublic<PublicForm>(path(slug));
  if (!res.data) return <PublicProblem kind={res.error} message={res.message} />;
  const form = res.data;
  return (
    <>
      <div className="w-full max-w-xl">
        <h1 className="text-[28px] leading-tight font-semibold">{form.name}</h1>
        {form.description ? <p className="mt-2 text-sm text-muted-foreground">{form.description}</p> : null}
        <Card className="mt-6 p-(--card-pad)">
          <PublicLeadForm form={form} />
        </Card>
      </div>
      <ChatBubble />
    </>
  );
}
