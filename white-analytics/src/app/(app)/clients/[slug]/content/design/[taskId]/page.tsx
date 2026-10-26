import type { Metadata } from "next";
import Link from "next/link";
import { ArrowLeft, ExternalLink } from "lucide-react";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { dayLabel } from "@/features/publishing/time";
import { DesignTaskActions, DesignVersions } from "@/features/content/components/design/task-detail";
import { ContentOffline } from "@/features/content/components/offline";
import { DesignStatusBadge } from "@/features/content/components/status";
import { contentContext, getDesignTask, listSizePresets, loadContent } from "@/features/content/queries";
import { c } from "@/features/content/strings";
import { formatRelative, initials } from "@/lib/format";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]/content/design/[taskId]">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await contentContext(slug);
  return { title: `${client.name} · ${c.taskEyebrow}` };
}

export default async function DesignTaskPage(props: PageProps<"/clients/[slug]/content/design/[taskId]">) {
  const { slug, taskId } = await props.params;
  const { client, canWrite, people, names } = await contentContext(slug);
  const base = `/clients/${slug}/content`;
  const back = (
    <Button asChild variant="outline" size="sm" className="group/nudge">
      <Link href={`${base}/design`}>
        <ArrowLeft className="nudge nudge-back size-3.5" /> {t.nav.contentDesign}
      </Link>
    </Button>
  );
  const res = await loadContent(() => Promise.all([getDesignTask(client.id, taskId), listSizePresets()]));
  if (!res.ok) {
    return (
      <>
        <PageHeader eyebrow={c.taskEyebrow} title={c.designTitle} actions={back} />
        <ContentOffline message={res.message} />
      </>
    );
  }

  const [{ task, files = [], feedback = [] }, presets] = res.data;
  const preset = presets.find((p) => p.key === task.sizePreset);

  return (
    <>
      <PageHeader
        eyebrow={c.taskEyebrow}
        title={task.title}
        description={`${preset?.label ?? task.sizePreset} · ${task.width}×${task.height} · ${c.designer}: ${task.designerId ? (names[task.designerId] ?? c.none) : c.unassigned}`}
        actions={
          <>
            <DesignStatusBadge status={task.status} className="h-6 px-2.5 text-xs" />
            <Button asChild variant="outline" size="sm">
              <Link href={`${base}/board/${task.itemId}`}>{c.openItem}</Link>
            </Button>
            {back}
          </>
        }
      />

      {canWrite ? <DesignTaskActions slug={slug} clientId={client.id} task={task} people={people} presets={presets} /> : null}

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_380px]">
        <div className="min-w-0 space-y-6">
          <Section index="01" title={c.versions}>
            <DesignVersions key={task.currentVersion} slug={slug} clientId={client.id} task={task} files={files} canWrite={canWrite} />
          </Section>
        </div>
        <aside className="min-w-0 space-y-6">
          <Section index="02" title={c.taskBrief}>
            <dl className="space-y-3 rounded-[1.5rem] bg-card p-6 text-sm shadow-(--card-shadow)">
              <div>
                <dt className="label-mono text-muted-foreground">{c.taskBrief}</dt>
                <dd className="mt-0.5 whitespace-pre-wrap">{task.brief || c.none}</dd>
              </div>
              <div>
                <dt className="label-mono text-muted-foreground">{c.dueDate}</dt>
                <dd className="mt-0.5">{task.dueDate ? dayLabel(task.dueDate, true) : c.none}</dd>
              </div>
              <div>
                <dt className="label-mono text-muted-foreground">{c.references}</dt>
                <dd className="mt-0.5 space-y-1">
                  {(task.references ?? []).length === 0
                    ? c.none
                    : task.references!.map((r) => (
                        <a key={r} href={r} target="_blank" rel="noreferrer" className="flex items-center gap-1 break-all text-brand underline-offset-4 hover:underline">
                          <ExternalLink className="size-3 shrink-0" /> {r}
                        </a>
                      ))}
                </dd>
              </div>
            </dl>
          </Section>
          <Section index="03" title={c.feedback}>
            <div className="rounded-[1.5rem] bg-card p-6 shadow-(--card-shadow)">
              {feedback.length === 0 ? (
                <p className="text-sm text-muted-foreground">{c.noFeedback}</p>
              ) : (
                <ul className="space-y-3">
                  {feedback.map((f) => (
                    <li key={f.id} className="flex gap-2.5">
                      <Avatar className="size-7 shrink-0">
                        <AvatarFallback className="text-[10px]">{initials(f.userName || "?")}</AvatarFallback>
                      </Avatar>
                      <div className="min-w-0 flex-1 rounded-lg bg-muted/50 px-3 py-2">
                        <p className="flex flex-wrap items-center gap-1.5 text-xs">
                          <span className="font-medium">{f.userName || c.system}</span>
                          <span className="text-muted-foreground">· {formatRelative(f.createdAt)}</span>
                          <Badge variant="secondary" className="h-4 px-1.5 text-[10px]">
                            {c.version(f.version)} · {c.decision[f.decision]}
                          </Badge>
                        </p>
                        {f.body ? <p className="mt-0.5 text-[13px] leading-snug whitespace-pre-wrap break-words">{f.body}</p> : null}
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </Section>
        </aside>
      </div>
    </>
  );
}
