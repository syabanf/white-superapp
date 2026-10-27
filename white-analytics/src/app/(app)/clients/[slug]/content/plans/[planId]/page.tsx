import type { Metadata } from "next";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { DetailLayout } from "@/components/dashboard/layout";
import { BriefForm } from "@/features/content/components/plans/brief-form";
import { PlanActions } from "@/features/content/components/plans/plan-actions";
import { PlanItems } from "@/features/content/components/plans/plan-items";
import { CommentThread } from "@/features/content/components/comment-thread";
import { ContentOffline } from "@/features/content/components/offline";
import { PlanStatusBadge } from "@/features/content/components/status";
import { contentContext, getPlan, loadContent } from "@/features/content/queries";
import { c } from "@/features/content/strings";
import { monthLabel, todayIn } from "@/features/publishing/time";
import { formatDateTime } from "@/lib/format";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]/content/plans/[planId]">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await contentContext(slug);
  return { title: `${client.name} · ${t.nav.contentPlans}` };
}

export default async function PlanDetailPage(props: PageProps<"/clients/[slug]/content/plans/[planId]">) {
  const { slug, planId } = await props.params;
  const { client, role, canWrite, userId, people, names } = await contentContext(slug);
  const back = { href: `/clients/${slug}/content/plans`, label: t.nav.contentPlans };
  const res = await loadContent(() => getPlan(client.id, planId));
  if (!res.ok) {
    return (
      <>
        <PageHeader eyebrow={t.nav.contentPlans} title={c.plansTitle} back={back} />
        <ContentOffline message={res.message} />
      </>
    );
  }

  const { plan, items = [], comments = [] } = res.data;
  const tz = client.timezone;
  const nameOf = (id?: string) => (id ? (names[id] ?? c.none) : c.none);
  const meta = [
    plan.submittedAt ? c.submittedBy(nameOf(plan.submittedBy), formatDateTime(plan.submittedAt, tz)) : null,
    plan.approvedAt ? c.approvedBy(nameOf(plan.approvedBy), formatDateTime(plan.approvedAt, tz)) : null,
  ].filter(Boolean);

  return (
    <>
      <PageHeader
        back={back}
        eyebrow={`${t.nav.contentPlans} · ${monthLabel(plan.month)}`}
        title={plan.title}
        description={meta.length ? meta.join(" · ") : c.briefDesc}
        actions={<PlanStatusBadge status={plan.status} className="h-6 px-2.5 text-xs" />}
      />

      {plan.reviewNote && plan.status === "PLAN_DRAFT" ? (
        <p className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm">
          <span className="label-mono mr-2 text-destructive">{c.reviewNote}</span>
          {plan.reviewNote}
        </p>
      ) : null}

      {canWrite && plan.status !== "PLAN_APPROVED" ? (
        <PlanActions
          slug={slug}
          planId={plan.id}
          status={plan.status}
          ownSubmission={role !== "ADMIN" && plan.submittedBy === userId}
        />
      ) : null}

      <DetailLayout
        aside={
          <>
            <Section index="02" title={c.brief} description={c.briefDesc}>
              <BriefForm key={plan.updatedAt} slug={slug} plan={plan} editable={canWrite && plan.status === "PLAN_DRAFT"} />
              {plan.generatedAt && plan.aiModel ? (
                <p className="text-xs text-muted-foreground">{c.generatedBy(plan.aiModel, formatDateTime(plan.generatedAt, tz))}</p>
              ) : null}
            </Section>
            <Section index="03" title={c.comments}>
              <div className="rounded-[1.5rem] bg-card p-(--card-pad) shadow-(--card-shadow)">
                <CommentThread slug={slug} target={{ kind: "plan", planId: plan.id }} comments={comments} />
              </div>
            </Section>
          </>
        }
      >
        <Section index="01" title={c.itemsSection}>
          <PlanItems
            slug={slug}
            planId={plan.id}
            month={plan.month}
            items={items}
            people={people}
            pillars={plan.brief?.pillars ?? []}
            timezone={tz}
            today={todayIn(tz)}
            canWrite={canWrite}
          />
        </Section>
      </DetailLayout>
    </>
  );
}
