"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Check, Loader2, Send, Sparkles, Trash2, Undo2 } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Toolbar } from "@/components/dashboard/layout";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  approvePlan,
  deletePlan,
  generatePlanDraft,
  requestPlanChanges,
  submitPlan,
} from "@/features/content/actions";
import { c } from "@/features/content/strings";
import type { PlanStatus } from "@/features/content/types";

type Props = {
  slug: string;
  planId: string;
  status: PlanStatus;
  /** The caller submitted this plan and is not an admin: approval belongs to someone else. */
  ownSubmission: boolean;
};

type DialogKind = "generate" | "changes" | "delete" | null;

/** Plan workflow buttons. Rendered only for members who may write. */
export function PlanActions({ slug, planId, status, ownSubmission }: Props) {
  const router = useRouter();
  const [pending, start] = React.useTransition();
  const [dialog, setDialog] = React.useState<DialogKind>(null);
  const [replace, setReplace] = React.useState(false);
  const [note, setNote] = React.useState("");

  const run = (fn: () => Promise<{ ok: true } | { ok: false; error: string }>, success: string, after?: () => void) =>
    start(async () => {
      const res = await fn();
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      toast.success(success);
      setDialog(null);
      if (after) after();
      else router.refresh();
    });

  return (
    <>
      <Toolbar label={c.planActions}>
        {status === "PLAN_DRAFT" ? (
          <>
            <Button variant="outline" disabled={pending} onClick={() => setDialog("generate")}>
              <Sparkles /> {c.generate}
            </Button>
            <Button disabled={pending} onClick={() => run(() => submitPlan({ slug, planId }), c.submitted)}>
              <Send /> {c.submit}
            </Button>
          </>
        ) : null}
        {status === "PLAN_IN_REVIEW" ? (
          <>
            <Button disabled={pending || ownSubmission} onClick={() => run(() => approvePlan({ slug, planId }), c.approved)}>
              <Check /> {c.approve}
            </Button>
            <Button variant="outline" disabled={pending} onClick={() => setDialog("changes")}>
              <Undo2 /> {c.requestChanges}
            </Button>
            {ownSubmission ? <span className="text-xs text-muted-foreground">{c.ownApprovalHint}</span> : null}
          </>
        ) : null}
        {status === "PLAN_DRAFT" ? (
          <Button variant="destructive" className="ml-auto" disabled={pending} onClick={() => setDialog("delete")}>
            <Trash2 /> {c.deletePlan}
          </Button>
        ) : null}
        {pending ? <Loader2 className="size-4 animate-spin text-muted-foreground" /> : null}
      </Toolbar>

      <Dialog open={dialog === "generate"} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{c.generateTitle}</DialogTitle>
            <DialogDescription>{c.generateDesc}</DialogDescription>
          </DialogHeader>
          <div className="flex items-start gap-2.5">
            <Checkbox id="gen-replace" checked={replace} onCheckedChange={(v) => setReplace(v === true)} className="mt-0.5" />
            <div className="space-y-0.5">
              <Label htmlFor="gen-replace">{c.replaceIdeas}</Label>
              <p className="text-xs text-muted-foreground">{c.replaceIdeasHint}</p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialog(null)}>
              {c.cancel}
            </Button>
            <Button disabled={pending} onClick={() => run(() => generatePlanDraft({ slug, planId, replace }), c.generated)}>
              {pending ? <Loader2 className="size-3.5 animate-spin" /> : <Sparkles className="size-3.5" />} {c.generate}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={dialog === "changes"} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{c.requestChangesTitle}</DialogTitle>
            <DialogDescription>{c.requestChangesDesc}</DialogDescription>
          </DialogHeader>
          <Textarea value={note} onChange={(e) => setNote(e.target.value)} placeholder={c.notePlaceholder} rows={3} autoFocus />
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialog(null)}>
              {c.cancel}
            </Button>
            <Button disabled={pending || !note.trim()} onClick={() => run(() => requestPlanChanges({ slug, planId, note }), c.changesRequested)}>
              {c.requestChanges}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={dialog === "delete"} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{c.deletePlan}</DialogTitle>
            <DialogDescription>{c.deletePlanDesc}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialog(null)}>
              {c.cancel}
            </Button>
            <Button
             
              variant="destructive"
              disabled={pending}
              onClick={() => run(() => deletePlan({ slug, planId }), c.planDeleted, () => router.push(`/clients/${slug}/content/plans`))}
            >
              {c.delete}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
