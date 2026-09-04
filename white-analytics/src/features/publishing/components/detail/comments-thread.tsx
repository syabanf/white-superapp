"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Loader2, SendHorizonal } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { addPostComment } from "@/features/publishing/actions";
import type { PostCommentView } from "@/features/publishing/queries";
import { p } from "@/features/publishing/strings";
import { formatRelative, initials } from "@/lib/format";

export function CommentsThread({ postId, comments }: { postId: string; comments: PostCommentView[] }) {
  const router = useRouter();
  const [body, setBody] = React.useState("");
  const [pending, start] = React.useTransition();
  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!body.trim()) return;
    start(async () => {
      const res = await addPostComment({ postId, body });
      if (res.ok) {
        setBody("");
        toast.success(p.toast.commentAdded);
        router.refresh();
      } else toast.error(res.error);
    });
  };
  return (
    <div className="space-y-3">
      {comments.length === 0 ? (
        <p className="text-sm text-muted-foreground">{p.noComments}</p>
      ) : (
        <ul className="space-y-3">
          {comments.map((c) => (
            <li key={c.id} className="flex gap-2.5">
              <Avatar className="size-7 shrink-0">
                <AvatarFallback className="text-[10px]">{initials(c.user ?? "?")}</AvatarFallback>
              </Avatar>
              <div className="min-w-0 flex-1 rounded-lg bg-muted/50 px-3 py-2">
                <p className="text-xs">
                  <span className="font-medium">{c.user ?? p.system}</span>
                  <span className="text-muted-foreground"> · {formatRelative(c.createdAt)}</span>
                </p>
                <p className="mt-0.5 text-[13px] leading-snug whitespace-pre-wrap break-words">{c.body}</p>
              </div>
            </li>
          ))}
        </ul>
      )}
      <form onSubmit={submit} className="space-y-2">
        <Textarea value={body} onChange={(e) => setBody(e.target.value)} placeholder={p.commentPlaceholder} rows={2} className="text-sm" disabled={pending} />
        <div className="flex justify-end">
          <Button type="submit" size="xs" disabled={pending || !body.trim()}>
            {pending ? <Loader2 className="size-3.5 animate-spin" /> : <SendHorizonal className="size-3.5" />} {p.send}
          </Button>
        </div>
      </form>
    </div>
  );
}
