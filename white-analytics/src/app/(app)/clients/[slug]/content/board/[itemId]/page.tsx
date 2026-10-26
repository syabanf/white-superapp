import type { Metadata } from "next";
import Link from "next/link";
import { ArrowLeft, ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { PlatformIcon, platformLabel } from "@/features/social/components/platform-icon";
import { dayLabel, dayLabelLong } from "@/features/publishing/time";
import { ActivityLog } from "@/features/content/components/activity-log";
import { CaptionEditor } from "@/features/content/components/board/caption-editor";
import { ItemActions } from "@/features/content/components/board/item-actions";
import { CommentThread } from "@/features/content/components/comment-thread";
import { ContentOffline } from "@/features/content/components/offline";
import { DesignStatusBadge, ItemStatusBadge } from "@/features/content/components/status";
import { contentContext, getItem, getPlan, listSizePresets, loadContent } from "@/features/content/queries";
import { c } from "@/features/content/strings";
import { formatDateTime } from "@/lib/format";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]/content/board/[itemId]">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await contentContext(slug);
  return { title: `${client.name} · ${c.itemEyebrow}` };
}

export default async function ContentItemPage(props: PageProps<"/clients/[slug]/content/board/[itemId]">) {
  const { slug, itemId } = await props.params;
  const { client, canWrite, people, names } = await contentContext(slug);
  const base = `/clients/${slug}/content`;
  const back = (
    <Button asChild variant="outline" size="sm" className="group/nudge">
      <Link href={`${base}/board`}>
        <ArrowLeft className="nudge nudge-back size-3.5" /> {t.nav.contentBoard}
      </Link>
    </Button>
  );
  const res = await loadContent(async () => {
    const [detail, presets] = await Promise.all([getItem(client.id, itemId), canWrite ? listSizePresets() : []]);
    const plan = canWrite ? await getPlan(client.id, detail.item.planId) : null;
    return { detail, presets, pillars: plan?.plan.brief?.pillars ?? [] };
  });
  if (!res.ok) {
    return (
      <>
        <PageHeader eyebrow={c.itemEyebrow} title={t.nav.contentBoard} actions={back} />
        <ContentOffline message={res.message} />
      </>
    );
  }

  const { detail, presets, pillars } = res.data;
  const { item, revisions = [], comments = [], activities = [], designTasks = [], allowedMoves = [] } = detail;
  const tz = client.timezone;
  const nameOf = (id?: string) => (id ? (names[id] ?? c.none) : c.unassigned);
  const editable = canWrite && item.status !== "SCHEDULED" && item.status !== "PUBLISHED";
  const details: [string, React.ReactNode][] = [
    [
      c.platform,
      <span key="p" className="inline-flex items-center gap-1.5">
        <PlatformIcon platform={item.platform} className="size-3.5" /> {platformLabel(item.platform)} · {c.format[item.format]}
      </span>,
    ],
    [c.pillar, item.pillar || c.none],
    [c.plannedDate, dayLabelLong(item.plannedDate)],
    [c.publishAt, item.publishAt ? formatDateTime(item.publishAt, tz) : c.none],
    [c.copywriter, nameOf(item.copywriterId)],
    [c.designer, nameOf(item.designerId)],
    [c.copyDue, item.copyDue ? dayLabel(item.copyDue, true) : c.none],
    [c.designDue, item.designDue ? dayLabel(item.designDue, true) : c.none],
    [c.reviewDue, item.reviewDue ? dayLabel(item.reviewDue, true) : c.none],
    [c.revisionsTotal, item.revisionRound > 0 ? c.round(item.revisionRound) : c.none],
  ];

  return (
    <>
      <PageHeader
        eyebrow={c.itemEyebrow}
        title={item.title}
        description={`${platformLabel(item.platform)} · ${c.format[item.format]} · ${dayLabelLong(item.plannedDate)}`}
        actions={
          <>
            <ItemStatusBadge status={item.status} className="h-6 px-2.5 text-xs" />
            <Button asChild variant="outline" size="sm">
              <Link href={`${base}/plans/${item.planId}`}>{t.nav.contentPlans}</Link>
            </Button>
            {back}
          </>
        }
      />

      {canWrite ? (
        <ItemActions slug={slug} item={item} allowedMoves={allowedMoves} people={people} pillars={pillars} presets={presets} timezone={tz} />
      ) : null}

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_380px]">
        <div className="min-w-0 space-y-6">
          <Section index="01" title={c.captionEditor} description={c.captionEditorDesc}>
            <CaptionEditor key={item.captionVersion} slug={slug} item={item} revisions={revisions} editable={editable} />
          </Section>

          <Section index="02" title={c.designTasks}>
            <div className="rounded-[1.5rem] bg-card p-2 shadow-(--card-shadow)">
              {designTasks.length === 0 ? (
                <p className="px-4 py-4 text-sm text-muted-foreground">{c.noDesignTasks}</p>
              ) : (
                <ul className="divide-y">
                  {designTasks.map((task) => (
                    <li key={task.id}>
                      <Link href={`${base}/design/${task.id}`} className="flex flex-wrap items-center gap-x-3 gap-y-1 rounded-xl px-4 py-3 hover:bg-muted/50">
                        <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{task.title}</span>
                        <span className="tabular text-xs text-muted-foreground">
                          {task.width}×{task.height} · {c.version(task.currentVersion)}
                        </span>
                        <DesignStatusBadge status={task.status} />
                      </Link>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </Section>

          <Section index="03" title={c.activityLog}>
            <div className="rounded-[1.5rem] bg-card p-6 shadow-(--card-shadow)">
              <ActivityLog activities={activities} timezone={tz} />
            </div>
          </Section>
        </div>

        <aside className="min-w-0 space-y-6">
          <Section index="04" title={c.details}>
            <dl className="grid grid-cols-2 gap-x-4 gap-y-3 rounded-[1.5rem] bg-card p-6 text-sm shadow-(--card-shadow)">
              {details.map(([label, value]) => (
                <div key={label}>
                  <dt className="label-mono text-muted-foreground">{label}</dt>
                  <dd className="mt-0.5">{value}</dd>
                </div>
              ))}
              <div className="col-span-2">
                <dt className="label-mono text-muted-foreground">{c.visualBrief}</dt>
                <dd className="mt-0.5 whitespace-pre-wrap">{item.visualBrief || c.none}</dd>
              </div>
            </dl>
          </Section>

          <Section index="05" title={c.publishResult}>
            <div className="space-y-2 rounded-[1.5rem] bg-card p-6 text-sm shadow-(--card-shadow)">
              {item.publishingPostId ? (
                <>
                  <p>
                    <span className="label-mono mr-2 text-muted-foreground">{c.publishingPost}</span>
                    <span className="font-mono text-xs">{item.publishingPostId}</span>
                  </p>
                  {item.publishedAt ? (
                    <p>
                      <span className="label-mono mr-2 text-muted-foreground">{c.publishedAt}</span>
                      {formatDateTime(item.publishedAt, tz)}
                    </p>
                  ) : null}
                  {item.permalink ? (
                    <a href={item.permalink} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1 break-all text-brand underline-offset-4 hover:underline">
                      <ExternalLink className="size-3 shrink-0" /> {c.permalink}
                    </a>
                  ) : null}
                  {item.publishError ? (
                    <p className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-xs">
                      <span className="label-mono mr-2 text-destructive">{c.publishError}</span>
                      {item.publishError}
                    </p>
                  ) : null}
                </>
              ) : (
                <p className="text-muted-foreground">{c.notScheduled}</p>
              )}
              <p className="flex flex-wrap gap-1 pt-1">
                <Badge variant={item.hasFinalDesign ? "secondary" : "outline"}>{item.hasFinalDesign ? c.finalDesign : c.noFinalDesign}</Badge>
              </p>
            </div>
          </Section>

          <Section index="06" title={c.comments}>
            <div className="rounded-[1.5rem] bg-card p-6 shadow-(--card-shadow)">
              <CommentThread
                slug={slug}
                target={{ kind: "item", itemId: item.id }}
                comments={comments.map((m) => ({
                  ...m,
                  tag: c.commentKind[m.kind] ? `${c.commentKind[m.kind]} · ${c.round(m.round)}` : undefined,
                }))}
              />
            </div>
          </Section>
        </aside>
      </div>
    </>
  );
}
