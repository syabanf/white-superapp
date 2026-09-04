import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { ArrowLeft, ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { DemoBanner } from "@/components/dashboard/banners";
import { Composer } from "@/features/publishing/components/composer/composer";
import { PostPreview } from "@/features/publishing/components/composer/preview";
import { PostActionBar } from "@/features/publishing/components/detail/action-bar";
import { CommentsThread } from "@/features/publishing/components/detail/comments-thread";
import { ActivityTimeline, TargetsList } from "@/features/publishing/components/detail/targets-and-timeline";
import { PostStatusBadge } from "@/features/publishing/components/post-status";
import { buildUtmUrl, canEdit, captionFor, PLATFORMS } from "@/features/publishing/lib";
import { getBestTimes, getOwnAccounts, getPostDetail, listMedia } from "@/features/publishing/queries";
import { todayIn } from "@/features/publishing/time";
import { p } from "@/features/publishing/strings";
import { isPublisherConfigured } from "@/lib/providers/social-publisher";
import { requireClientAccess, requireUser } from "@/lib/rbac";
import { formatDateTime } from "@/lib/format";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]/publish/posts/[id]">): Promise<Metadata> {
  const { slug, id } = await props.params;
  const { client } = await requireClientAccess(slug);
  const post = await getPostDetail(client.id, id);
  return { title: `${client.name} · ${post?.title || t.nav.posts}` };
}

export default async function PostDetailPage(props: PageProps<"/clients/[slug]/publish/posts/[id]">) {
  const [{ slug, id }, sp] = await Promise.all([props.params, props.searchParams]);
  const [{ client, role }, user] = await Promise.all([requireClientAccess(slug), requireUser()]);
  const post = await getPostDetail(client.id, id);
  if (!post) notFound();
  const tz = client.timezone;
  const base = `/clients/${slug}/publish`;
  const isAuthor = post.createdById === user.id;
  const editing = (Array.isArray(sp.edit) ? sp.edit[0] : sp.edit) === "1" && canEdit(post.status, role);
  const demo = !isPublisherConfigured() ? <DemoBanner message={p.demoBanner} settingsHref={`/clients/${slug}/settings`} /> : null;

  if (editing) {
    const [accounts, library, bestTimes] = await Promise.all([getOwnAccounts(client.id), listMedia(client.id), getBestTimes(client.id, tz)]);
    return (
      <>
        <PageHeader
          eyebrow={p.editing}
          title={post.title || p.untitled}
          description={p.composeSubtitle}
          actions={
            <Button asChild variant="outline" size="sm" className="group/nudge">
              <Link href={`${base}/posts/${post.id}`}>
                <ArrowLeft className="nudge nudge-back size-3.5" /> {p.backToDetail}
              </Link>
            </Button>
          }
        />
        {demo}
        <Composer clientId={client.id} slug={slug} timezone={tz} today={todayIn(tz)} accounts={accounts} library={library} bestTimes={bestTimes} initial={post} role={role} />
      </>
    );
  }

  const bestTimes = role === "VIEWER" ? [] : await getBestTimes(client.id, tz);
  const platforms = PLATFORMS.filter((pl) => post.targets.some((tg) => tg.platform === pl));
  const when = post.publishedAt
    ? `${p.publishedAt} ${formatDateTime(post.publishedAt, tz)}`
    : post.scheduledAt
      ? `${post.status === "SCHEDULED" ? p.scheduledFor : p.planned} ${formatDateTime(post.scheduledAt, tz)}`
      : p.noSchedule;
  const finalLink = post.linkUrl ? buildUtmUrl(post.linkUrl, post.utm) : null;

  return (
    <>
      <PageHeader
        eyebrow={p.detailEyebrow}
        title={post.title || p.untitled}
        description={
          <>
            {when} · {p.author}: {post.createdBy ?? "–"}
            {post.approvedBy ? ` · ${p.approver}: ${post.approvedBy}` : ""}
          </>
        }
        actions={
          <>
            <PostStatusBadge status={post.status} className="h-6 px-2.5 text-xs" />
            <Button asChild variant="outline" size="sm" className="group/nudge">
              <Link href={`${base}/posts`}>
                <ArrowLeft className="nudge nudge-back size-3.5" /> {t.nav.posts}
              </Link>
            </Button>
          </>
        }
      />
      {demo}
      {post.reviewNote && post.status === "REJECTED" ? (
        <p className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm">
          <span className="label-mono mr-2 text-destructive">{p.reviewNote}</span>
          {post.reviewNote}
        </p>
      ) : null}

      <div className="hidden lg:block">
        <PostActionBar postId={post.id} status={post.status} scheduledAt={post.scheduledAt} slug={slug} role={role} isAuthor={isAuthor} timezone={tz} today={todayIn(tz)} bestTimes={bestTimes} hasTargets={post.targets.length > 0} />
      </div>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_380px]">
        <div className="min-w-0 space-y-6">
          <Section index="01" title={p.sectionPreview}>
            {platforms.length === 0 ? (
              <p className="rounded-xl border border-dashed px-3 py-6 text-center text-sm text-muted-foreground">{p.previewEmpty}</p>
            ) : (
              <div className="grid gap-4 sm:grid-cols-2">
                {platforms.map((pl) => {
                  const tg = post.targets.find((x) => x.platform === pl)!;
                  return (
                    <PostPreview
                      key={pl}
                      platform={pl}
                      account={{ username: tg.username, displayName: tg.displayName, avatarUrl: tg.avatarUrl }}
                      caption={captionFor(post.body, tg.bodyOverride)}
                      media={post.media}
                      linkUrl={finalLink ?? undefined}
                    />
                  );
                })}
              </div>
            )}
          </Section>

          <Section index="02" title={p.metaTitle}>
            <dl className="grid gap-x-6 gap-y-3 rounded-xl border bg-card p-4 text-sm sm:grid-cols-2">
              <div>
                <dt className="label-mono text-muted-foreground">{p.linkOut}</dt>
                <dd className="mt-1 break-all">
                  {finalLink ? (
                    <a href={finalLink} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1 text-brand underline-offset-4 hover:underline">
                      <ExternalLink className="size-3 shrink-0" /> {finalLink}
                    </a>
                  ) : (
                    <span className="text-muted-foreground">{p.noLink}</span>
                  )}
                </dd>
              </div>
              <div>
                <dt className="label-mono text-muted-foreground">{p.labels}</dt>
                <dd className="mt-1 flex flex-wrap gap-1">
                  {post.labels.length ? post.labels.map((l) => <Badge key={l} variant="secondary">{l}</Badge>) : <span className="text-muted-foreground">{p.labelsEmpty}</span>}
                </dd>
              </div>
              {post.firstComment ? (
                <div className="sm:col-span-2">
                  <dt className="label-mono text-muted-foreground">{p.firstComment}</dt>
                  <dd className="mt-1 whitespace-pre-wrap">{post.firstComment}</dd>
                </div>
              ) : null}
              <div className="sm:col-span-2">
                <dt className="label-mono text-muted-foreground">{p.sectionCaption}</dt>
                <dd className="mt-1 whitespace-pre-wrap break-words leading-relaxed">{post.body || "–"}</dd>
              </div>
            </dl>
          </Section>

          <Section index="03" title={p.timeline}>
            <div className="rounded-xl border bg-card p-4">
              <ActivityTimeline activities={post.activities} timezone={tz} />
            </div>
          </Section>
        </div>

        <aside className="min-w-0 space-y-6">
          <Section index="04" title={p.targets}>
            <TargetsList targets={post.targets} timezone={tz} />
          </Section>
          <Section index="05" title={p.comments}>
            <div className="rounded-xl border bg-card p-4">
              <CommentsThread postId={post.id} comments={post.comments} />
            </div>
          </Section>
        </aside>
      </div>

      <div className="lg:hidden">
        <PostActionBar postId={post.id} status={post.status} scheduledAt={post.scheduledAt} slug={slug} role={role} isAuthor={isAuthor} timezone={tz} today={todayIn(tz)} bestTimes={bestTimes} hasTargets={post.targets.length > 0} />
      </div>
    </>
  );
}
