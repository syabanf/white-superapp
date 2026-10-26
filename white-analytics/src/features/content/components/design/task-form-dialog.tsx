"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { createDesignTask, updateDesignTask } from "@/features/content/actions";
import { c } from "@/features/content/strings";
import type { DesignStatus, DesignTaskInput, Person, SizePreset } from "@/features/content/types";
import { Field, PersonSelect } from "../fields";

const DEFAULT = "__default";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  slug: string;
  people: Person[];
  presets: SizePreset[];
  mode:
    | { kind: "create"; itemId: string; initial: DesignTaskInput }
    | { kind: "edit"; taskId: string; initial: DesignTaskInput; status: DesignStatus };
};

export function DesignTaskFormDialog({ open, onOpenChange, slug, people, presets, mode }: Props) {
  const router = useRouter();
  const [pending, start] = React.useTransition();
  const [v, setV] = React.useState(mode.initial);
  const [refs, setRefs] = React.useState(mode.initial.references.join("\n"));
  const set = <K extends keyof DesignTaskInput>(k: K, val: DesignTaskInput[K]) => setV((s) => ({ ...s, [k]: val }));

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const task: DesignTaskInput = {
      ...v,
      references: refs
        .split("\n")
        .map((r) => r.trim())
        .filter(Boolean),
    };
    start(async () => {
      const res =
        mode.kind === "create"
          ? await createDesignTask({ slug, itemId: mode.itemId, task })
          : await updateDesignTask({ slug, taskId: mode.taskId, task, status: mode.status });
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      toast.success(mode.kind === "create" ? c.taskCreated : c.taskSaved);
      onOpenChange(false);
      router.refresh();
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{mode.kind === "create" ? c.taskNewTitle : c.taskEditTitle}</DialogTitle>
          <DialogDescription className="sr-only">{c.designDesc}</DialogDescription>
        </DialogHeader>
        <form id="task-form" onSubmit={submit} className="grid gap-4 sm:grid-cols-2">
          <Field id="tk-title" label={c.titleLabel} className="sm:col-span-2">
            <Input id="tk-title" value={v.title} onChange={(e) => set("title", e.target.value)} autoFocus />
          </Field>
          <Field id="tk-brief" label={c.taskBrief} className="sm:col-span-2">
            <Textarea id="tk-brief" rows={3} value={v.brief} onChange={(e) => set("brief", e.target.value)} />
          </Field>
          <Field id="tk-refs" label={c.references} hint={c.referencesHint} className="sm:col-span-2">
            <Textarea id="tk-refs" rows={2} value={refs} onChange={(e) => setRefs(e.target.value)} />
          </Field>
          <Field id="tk-size" label={c.sizePreset} className="sm:col-span-2">
            <Select value={v.sizePreset || DEFAULT} onValueChange={(x) => set("sizePreset", x === DEFAULT ? "" : x)}>
              <SelectTrigger id="tk-size" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={DEFAULT}>{c.sizeDefault}</SelectItem>
                {presets.map((p) => (
                  <SelectItem key={p.key} value={p.key}>
                    {p.label} · {p.width}×{p.height}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <Field id="tk-designer" label={c.designer}>
            <PersonSelect id="tk-designer" value={v.designerId} onChange={(x) => set("designerId", x)} people={people} />
          </Field>
          <Field id="tk-due" label={c.dueDate}>
            <Input id="tk-due" type="date" value={v.dueDate} onChange={(e) => set("dueDate", e.target.value)} />
          </Field>
        </form>
        <DialogFooter>
          <Button variant="outline" size="sm" onClick={() => onOpenChange(false)}>
            {c.cancel}
          </Button>
          <Button type="submit" form="task-form" size="sm" disabled={pending}>
            {pending ? <Loader2 className="size-3.5 animate-spin" /> : null} {c.save}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
