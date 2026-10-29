"use client";

import { CheckCheck, Loader2, Save, SendHorizonal, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import type { DraftStatus } from "@/features/campaigns/lib";
import { c } from "@/features/campaigns/strings";

export type PendingAction = "save" | "ready" | "submit" | "delete" | null;

/**
 * Sticky action bar. On phones it replaces the app tab bar (this route is listed
 * in the shell HAS_OWN_BOTTOM_BAR list); from lg it sits in flow.
 */
export function BuilderActions({
  status,
  canDelete,
  simulated,
  pending,
  onSave,
  onReady,
  onSubmit,
  onDelete,
}: {
  status: DraftStatus;
  canDelete: boolean;
  simulated: boolean;
  pending: PendingAction;
  onSave: () => void;
  onReady: () => void;
  onSubmit: () => void;
  onDelete: () => void;
}) {
  const busy = pending !== null;
  const spin = (a: PendingAction) => (pending === a ? <Loader2 className="animate-spin" /> : null);
  return (
    <div className="sticky bottom-[max(0.75rem,env(safe-area-inset-bottom))] z-10 flex items-center gap-2 rounded-[1.5rem] bg-card p-2.5 shadow-(--card-shadow) lg:static lg:p-3">
      {canDelete ? (
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button type="button" variant="ghost" size="icon-sm" aria-label={c.delete} disabled={busy}>
              {spin("delete") ?? <Trash2 />}
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>{c.deleteConfirmTitle}</AlertDialogTitle>
              <AlertDialogDescription>{c.deleteConfirmDesc}</AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>{c.cancel}</AlertDialogCancel>
              <AlertDialogAction variant="destructive" onClick={onDelete}>
                {c.confirmDelete}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      ) : null}
      {status !== "SUBMITTED" ? (
        <div className="ml-auto flex items-center gap-2">
          <Button type="button" variant="outline" size="sm" onClick={onSave} disabled={busy}>
            {spin("save") ?? <Save />}
            {pending === "save" ? c.saving : c.save}
          </Button>
          {status === "READY" ? (
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button type="button" size="sm" disabled={busy}>
                  {spin("submit") ?? <SendHorizonal />}
                  {pending === "submit" ? c.submitting : simulated ? c.submitSimulated : c.submit}
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>{c.submitConfirmTitle}</AlertDialogTitle>
                  <AlertDialogDescription>{simulated ? c.submitConfirmSim : c.submitConfirmReal}</AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>{c.cancel}</AlertDialogCancel>
                  <AlertDialogAction onClick={onSubmit}>{c.confirm}</AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          ) : (
            <Button type="button" size="sm" onClick={onReady} disabled={busy}>
              {spin("ready") ?? <CheckCheck />}
              {c.markReady}
            </Button>
          )}
        </div>
      ) : null}
    </div>
  );
}
