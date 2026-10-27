"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Loader2, SendHorizonal } from "lucide-react";
import { toast } from "@/lib/toast";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { addItemComment, addPlanComment } from "@/features/content/actions";
import { c } from "@/features/content/strings";
import { formatRelative, initials } from "@/lib/format";

export type ThreadComment = { id: string; userName: string; body: string; createdAt: string; tag?: string };

/** Comments on a plan or an item. Every member, VIEWER included, may comment. */
export function CommentThread({
  slug,
  target,
  comments,
}: {
  slug: string;
  target: { kind: "plan"; planId: string } | { kind: "item"; itemId: string };
  comments: ThreadComment[];
}) {
  const router = useRouter();
  const [body, setBody] = React.useState("");
  const [pending, start] = React.useTransition();

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!body.trim()) return;
    start(async () => {
      const res =
        target.kind === "plan"
          ? await addPlanComment({ slug, planId: target.planId, body })
          : await addItemComment({ slug, itemId: target.itemId, body });
      if (res.ok) {
        setBody("");
        toast.success(c.commentAdded);
        router.refresh();
      } else toast.error(res.error);
    });
  };

  return (
    <div className="space-y-3">
      {comments.length === 0 ? (
        <p className="text-sm text-muted-foreground">{c.noComments}</p>
      ) : (
        <ul className="space-y-3">
          {comments.map((m) => (
            <li key={m.id} className="flex gap-2.5">
              <Avatar className="size-7 shrink-0">
                <AvatarFallback className="text-[10px]">{initials(m.userName || "?")}</AvatarFallback>
              </Avatar>
              <div className="min-w-0 flex-1 rounded-lg bg-muted/50 px-3 py-2">
                <p className="flex flex-wrap items-center gap-1.5 text-xs">
                  <span className="font-medium">{m.userName || c.system}</span>
                  <span className="text-muted-foreground">· {formatRelative(m.createdAt)}</span>
                  {m.tag ? <Badge variant="secondary" className="h-4 px-1.5 text-[10px]">{m.tag}</Badge> : null}
                </p>
                <p className="mt-0.5 text-[13px] leading-snug whitespace-pre-wrap break-words">{m.body}</p>
              </div>
            </li>
          ))}
        </ul>
      )}
      <form onSubmit={submit} className="space-y-2">
        <Textarea value={body} onChange={(e) => setBody(e.target.value)} placeholder={c.commentPlaceholder} rows={2} className="text-sm" disabled={pending} />
        <div className="flex justify-end">
          <Button type="submit" disabled={pending || !body.trim()}>
            {pending ? <Loader2 className="animate-spin" /> : <SendHorizonal />} {c.send}
          </Button>
        </div>
      </form>
    </div>
  );
}
