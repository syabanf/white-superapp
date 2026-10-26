"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Check, Download, Loader2, MessageSquare, Pencil, Play, Undo2, Upload } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { reviewDesign, updateDesignTask } from "@/features/content/actions";
import { contentFiles, formatBytes } from "@/features/content/lib";
import { c } from "@/features/content/strings";
import type { DesignFile, DesignStatus, DesignTask, DesignTaskInput, Person, ReviewDecision, SizePreset } from "@/features/content/types";
import { cn } from "@/lib/utils";
import { FilePreview } from "../file-preview";
import { UploadDialog } from "../upload-dialog";
import { DesignTaskFormDialog } from "./task-form-dialog";

function inputOf(task: DesignTask): DesignTaskInput {
  return {
    title: task.title,
    brief: task.brief,
    references: task.references ?? [],
    sizePreset: task.sizePreset,
    designerId: task.designerId ?? "",
    dueDate: task.dueDate ?? "",
  };
}

/** Edit, manual status and version upload. Members who may write only. */
export function DesignTaskActions({
  slug,
  clientId,
  task,
  people,
  presets,
}: {
  slug: string;
  clientId: string;
  task: DesignTask;
  people: Person[];
  presets: SizePreset[];
}) {
  const router = useRouter();
  const [dialog, setDialog] = React.useState<"edit" | "upload" | null>(null);
  const [pending, start] = React.useTransition();

  const setStatus = (status: DesignStatus) =>
    start(async () => {
      const res = await updateDesignTask({ slug, taskId: task.id, task: inputOf(task), status });
      if (res.ok) {
        toast.success(c.taskSaved);
        router.refresh();
      } else toast.error(res.error);
    });

  return (
    <>
      <div className="flex flex-wrap items-center gap-2 rounded-xl border bg-card p-3">
        <Button size="sm" disabled={pending} onClick={() => setDialog("upload")}>
          <Upload className="size-3.5" /> {c.uploadVersion}
        </Button>
        {task.status === "DESIGN_TODO" ? (
          <Button size="sm" variant="outline" disabled={pending} onClick={() => setStatus("DESIGN_IN_PROGRESS")}>
            <Play className="size-3.5" /> {c.startWork}
          </Button>
        ) : null}
        {task.status === "DESIGN_IN_PROGRESS" ? (
          <Button size="sm" variant="ghost" disabled={pending} onClick={() => setStatus("DESIGN_TODO")}>
            <Undo2 className="size-3.5" /> {c.backToTodo}
          </Button>
        ) : null}
        <Button size="sm" variant="ghost" disabled={pending} onClick={() => setDialog("edit")}>
          <Pencil className="size-3.5" /> {c.edit}
        </Button>
        {pending ? <Loader2 className="size-4 animate-spin text-muted-foreground" /> : null}
      </div>

      {dialog === "edit" ? (
        <DesignTaskFormDialog
          open
          onOpenChange={(o) => !o && setDialog(null)}
          slug={slug}
          people={people}
          presets={presets}
          mode={{ kind: "edit", taskId: task.id, initial: inputOf(task), status: task.status }}
        />
      ) : null}
      <UploadDialog
        open={dialog === "upload"}
        onOpenChange={(o) => setDialog(o ? "upload" : null)}
        title={c.uploadVersion}
        description={c.uploadVersionDesc}
        url={contentFiles.uploadDesign(clientId, task.id)}
        fileField="files"
        multiple
        accept="image/*,video/*,application/pdf"
        noteLabel={c.versionNote}
        success={c.uploaded}
      />
    </>
  );
}

/** Version switcher with previews, downloads and the review form. */
export function DesignVersions({
  slug,
  clientId,
  task,
  files,
  canWrite,
}: {
  slug: string;
  clientId: string;
  task: DesignTask;
  files: DesignFile[];
  canWrite: boolean;
}) {
  const router = useRouter();
  const versions = React.useMemo(() => Array.from(new Set(files.map((f) => f.version))).sort((a, b) => b - a), [files]);
  const [version, setVersion] = React.useState(versions[0] ?? 0);
  const [fileId, setFileId] = React.useState<string | null>(null);
  const [body, setBody] = React.useState("");
  const [pending, start] = React.useTransition();
  const shown = files.filter((f) => f.version === version).sort((a, b) => a.sortOrder - b.sortOrder);
  const isLatest = version === task.currentVersion;

  const review = (decision: ReviewDecision) =>
    start(async () => {
      const res = await reviewDesign({ slug, taskId: task.id, decision, body, fileId: fileId ?? undefined });
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      toast.success(decision === "APPROVE" ? c.designApproved : decision === "REQUEST_CHANGES" ? c.designChanges : c.reviewSent);
      setBody("");
      setFileId(null);
      router.refresh();
    });

  if (files.length === 0) {
    return <p className="rounded-xl border border-dashed px-3 py-8 text-center text-sm text-muted-foreground">{c.noFiles}</p>;
  }

  return (
    <div className="space-y-4">
      {versions.length > 1 ? (
        <Tabs value={String(version)} onValueChange={(v) => setVersion(Number(v))}>
          <TabsList>
            {versions.map((v) => (
              <TabsTrigger key={v} value={String(v)}>
                {c.version(v)}
                {v === task.currentVersion ? <span className="ml-1 text-muted-foreground">· {c.current}</span> : null}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      ) : null}

      <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        {shown.map((f, i) => {
          const src = contentFiles.designFile(clientId, f.id);
          const selected = fileId === f.id;
          return (
            <li key={f.id} className={cn("overflow-hidden rounded-xl border bg-card", selected && "ring-2 ring-brand")}>
              <button
                type="button"
                onClick={() => setFileId(selected ? null : f.id)}
                className="block aspect-[4/5] w-full"
                aria-pressed={selected}
                aria-label={f.fileName}
              >
                <FilePreview src={src} mime={f.mimeType} name={f.fileName} />
              </button>
              <div className="flex items-center gap-1.5 border-t px-2 py-1.5 text-xs">
                <span className="tabular text-muted-foreground">{i + 1}</span>
                <span className="min-w-0 flex-1 truncate" title={f.fileName}>
                  {f.fileName}
                </span>
                <span className="tabular shrink-0 text-muted-foreground">{formatBytes(f.sizeBytes)}</span>
                <a href={src} download={f.fileName} className="shrink-0 rounded p-0.5 hover:bg-muted" aria-label={`${c.download} ${f.fileName}`}>
                  <Download className="size-3.5" />
                </a>
              </div>
            </li>
          );
        })}
      </ul>

      {isLatest ? (
        <div className="space-y-2 rounded-xl border bg-card p-3">
          <Textarea value={body} onChange={(e) => setBody(e.target.value)} placeholder={c.reviewPlaceholder} rows={2} className="text-sm" disabled={pending} />
          <div className="flex flex-wrap items-center justify-end gap-2">
            {fileId ? <span className="mr-auto text-xs text-muted-foreground">{shown.find((f) => f.id === fileId)?.fileName}</span> : null}
            <Button size="xs" variant="outline" disabled={pending || !body.trim()} onClick={() => review("COMMENT")}>
              <MessageSquare className="size-3.5" /> {c.decision.COMMENT}
            </Button>
            {canWrite && task.status === "DESIGN_IN_REVIEW" ? (
              <>
                <Button size="xs" variant="outline" disabled={pending || !body.trim()} onClick={() => review("REQUEST_CHANGES")}>
                  <Undo2 className="size-3.5" /> {c.decision.REQUEST_CHANGES}
                </Button>
                <Button size="xs" disabled={pending} onClick={() => review("APPROVE")}>
                  <Check className="size-3.5" /> {c.approve}
                </Button>
              </>
            ) : null}
            {pending ? <Loader2 className="size-4 animate-spin text-muted-foreground" /> : null}
          </div>
        </div>
      ) : null}
    </div>
  );
}
