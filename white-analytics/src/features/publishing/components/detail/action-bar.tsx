"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { CalendarClock, CalendarX, Check, Copy, Loader2, Pencil, RotateCcw, Send, SendHorizonal, Trash2, Undo2, X } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { approvePost, backToDraft, deletePost, duplicatePost, publishNow, rejectPost, retryPost, schedulePost, submitForReview, unschedulePost } from "@/features/publishing/actions";
import { canDelete, canEdit, canTransition, type PostRole, type PostStatus } from "@/features/publishing/lib";
import type { BestTimeSlot } from "@/features/publishing/queries";
import { fromZoned, toZoned } from "@/features/publishing/time";
import { SchedulePicker } from "@/features/publishing/components/composer/schedule-picker";
import { p } from "@/features/publishing/strings";
import { t } from "@/i18n/id";

type Props = {
  postId: string;
  status: PostStatus;
  scheduledAt: string | null;
  slug: string;
  role: PostRole;
  isAuthor: boolean;
  timezone: string;
  today: string;
  bestTimes: BestTimeSlot[];
  hasTargets: boolean;
};

type DialogKind = "reject" | "schedule" | "delete" | "publish" | null;

export function PostActionBar({ postId, status, scheduledAt, slug, role, isAuthor, timezone, today, bestTimes, hasTargets }: Props) {
  const router = useRouter();
  const base = `/clients/${slug}/publish`;
  const [pending, start] = React.useTransition();
  const [dialog, setDialog] = React.useState<DialogKind>(null);
  const [note, setNote] = React.useState("");
  const initial = scheduledAt ? toZoned(new Date(scheduledAt), timezone) : null;
  const [date, setDate] = React.useState(initial?.date ?? "");
  const [time, setTime] = React.useState(initial?.time ?? "");
  const ctx = { role, isAuthor };
  const can = (to: PostStatus) => canTransition(status, to, ctx);

  const run = (fn: () => Promise<{ ok: true; data: unknown } | { ok: false; error: string }>, success: string, after?: (data: unknown) => void) => {
    start(async () => {
      const res = await fn();
      if (res.ok) {
        toast.success(success);
        setDialog(null);
        if (after) after(res.data);
        else router.refresh();
      } else toast.error(res.error);
    });
  };

  const publishLabel = (data: unknown) => {
    const d = data as { status: string; failed: number } | undefined;
    if (d?.status === "PUBLISHED") toast.success(p.toast.published);
    else toast.warning(p.toast.publishFailed);
    router.refresh();
  };

  if (role === "VIEWER") return null;
  const backLabel = status === "IN_REVIEW" ? p.act.withdraw : status === "REJECTED" ? p.act.revise : p.act.backToDraft;

  return (
    <>
      <div className="sticky bottom-0 z-10 -mx-4 flex flex-wrap items-center gap-2 border-t bg-background/95 px-4 py-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] backdrop-blur md:-mx-6 md:px-6 lg:static lg:mx-0 lg:rounded-xl lg:border lg:bg-card lg:p-3 lg:backdrop-blur-none">
        {canEdit(status, role) ? (
          <Button asChild variant="outline" size="sm">
            <Link href={`${base}/posts/${postId}?edit=1`}>
              <Pencil className="size-3.5" /> {p.act.edit}
            </Link>
          </Button>
        ) : null}
        {can("IN_REVIEW") ? (
          <Button size="sm" variant={can("APPROVED") ? "outline" : "default"} disabled={pending} onClick={() => run(() => submitForReview(postId), p.toast.submitted)}>
            <Send className="size-3.5" /> {p.act.submit}
          </Button>
        ) : null}
        {can("APPROVED") && status !== "SCHEDULED" ? (
          <Button size="sm" disabled={pending} onClick={() => run(() => approvePost(postId), p.toast.approved)}>
            <Check className="size-3.5" /> {p.act.approve}
          </Button>
        ) : null}
        {can("REJECTED") ? (
          <Button size="sm" variant="outline" disabled={pending} onClick={() => setDialog("reject")}>
            <X className="size-3.5" /> {p.act.reject}
          </Button>
        ) : null}
        {can("SCHEDULED") ? (
          <Button size="sm" disabled={pending} onClick={() => setDialog("schedule")}>
            <CalendarClock className="size-3.5" /> {status === "FAILED" ? p.act.schedule : p.act.schedule}
          </Button>
        ) : null}
        {status === "SCHEDULED" && can("APPROVED") ? (
          <Button size="sm" variant="outline" disabled={pending} onClick={() => run(() => unschedulePost(postId), p.toast.unscheduled)}>
            <CalendarX className="size-3.5" /> {p.act.unschedule}
          </Button>
        ) : null}
        {can("PUBLISHING") && hasTargets ? (
          <Button size="sm" variant={status === "FAILED" ? "default" : "outline"} disabled={pending} onClick={() => setDialog("publish")}>
            {status === "FAILED" ? <RotateCcw className="size-3.5" /> : <SendHorizonal className="size-3.5" />}
            {status === "FAILED" ? p.act.retry : p.act.publishNow}
          </Button>
        ) : null}
        {can("DRAFT") ? (
          <Button size="sm" variant="ghost" disabled={pending} onClick={() => run(() => backToDraft(postId), t.common.saved)}>
            <Undo2 className="size-3.5" /> {backLabel}
          </Button>
        ) : null}
        <span className="ml-auto flex items-center gap-2">
          <Button
            size="sm"
            variant="ghost"
            disabled={pending}
            onClick={() => run(() => duplicatePost(postId), p.toast.duplicated, (d) => router.push(`${base}/posts/${(d as { id: string }).id}`))}
          >
            <Copy className="size-3.5" /> {p.act.duplicate}
          </Button>
          {canDelete(status, role) ? (
            <Button size="sm" variant="destructive" disabled={pending} onClick={() => setDialog("delete")}>
              <Trash2 className="size-3.5" /> {p.act.delete}
            </Button>
          ) : null}
        </span>
        {pending ? <Loader2 className="size-4 animate-spin text-muted-foreground" /> : null}
      </div>

      {/* reject */}
      <Dialog open={dialog === "reject"} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{p.rejectTitle}</DialogTitle>
            <DialogDescription>{p.rejectDesc}</DialogDescription>
          </DialogHeader>
          <Textarea value={note} onChange={(e) => setNote(e.target.value)} placeholder={p.rejectPlaceholder} rows={3} autoFocus />
          <DialogFooter>
            <Button variant="outline" size="sm" onClick={() => setDialog(null)}>
              {t.common.cancel}
            </Button>
            <Button size="sm" disabled={pending || !note.trim()} onClick={() => run(() => rejectPost({ postId, note }), p.toast.rejected)}>
              {p.act.reject}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* schedule */}
      <Dialog open={dialog === "schedule"} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{p.scheduleTitle}</DialogTitle>
            <DialogDescription>{p.scheduleDesc}</DialogDescription>
          </DialogHeader>
          <SchedulePicker
            date={date}
            time={time}
            min={today}
            timeZone={timezone}
            bestTimes={bestTimes}
            onChange={(d, tm) => {
              setDate(d);
              setTime(tm);
            }}
          />
          <DialogFooter>
            <Button variant="outline" size="sm" onClick={() => setDialog(null)}>
              {t.common.cancel}
            </Button>
            <Button
              size="sm"
              disabled={pending || !date || !time}
              onClick={() => run(() => schedulePost({ postId, scheduledAt: fromZoned(date, time, timezone).toISOString() }), p.toast.scheduled)}
            >
              <CalendarClock className="size-3.5" /> {p.act.schedule}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* publish now / retry */}
      <AlertDialog open={dialog === "publish"} onOpenChange={(o) => !o && setDialog(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{p.publishNowTitle}</AlertDialogTitle>
            <AlertDialogDescription>{p.publishNowDesc}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t.common.cancel}</AlertDialogCancel>
            <AlertDialogAction
              disabled={pending}
              onClick={(e) => {
                e.preventDefault();
                run(() => (status === "FAILED" ? retryPost(postId) : publishNow(postId)), p.toast.retrying, publishLabel);
              }}
            >
              {status === "FAILED" ? p.act.retry : p.act.publishNow}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* delete */}
      <AlertDialog open={dialog === "delete"} onOpenChange={(o) => !o && setDialog(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{p.deleteTitle}</AlertDialogTitle>
            <AlertDialogDescription>{p.deleteDesc}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t.common.cancel}</AlertDialogCancel>
            <AlertDialogAction
              disabled={pending}
              onClick={(e) => {
                e.preventDefault();
                run(() => deletePost(postId), p.toast.deleted, () => router.push(`${base}/posts`));
              }}
            >
              {t.common.delete}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
