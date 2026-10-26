"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ImageIcon, Loader2, MoreHorizontal } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { PlatformIcon } from "@/features/social/components/platform-icon";
import { dayLabel, monthLabel } from "@/features/publishing/time";
import { moveItem } from "@/features/content/actions";
import { proxyRequest } from "@/features/content/client";
import { contentFiles, currentDue, isOverdue, ITEM_STATUSES, ITEM_TONE, TONE_VAR } from "@/features/content/lib";
import { c } from "@/features/content/strings";
import type { Item, ItemDetail, ItemStatus, Person, Plan } from "@/features/content/types";
import { cn } from "@/lib/utils";
import { OptionSelect } from "../fields";
import { useSetParam } from "../use-set-param";

const DND_MIME = "application/x-content-item";

type Props = {
  slug: string;
  clientId: string;
  items: Item[];
  plans: Plan[];
  people: Person[];
  planId: string;
  assigneeId: string;
  today: string;
  canWrite: boolean;
};

export function ContentBoard({ slug, clientId, items, plans, people, planId, assigneeId, today, canWrite }: Props) {
  const router = useRouter();
  const [pending, start] = React.useTransition();
  const [dragging, setDragging] = React.useState<string | null>(null);
  // null while the allowed moves for the dragged card are loading
  const [allowed, setAllowed] = React.useState<ItemStatus[] | null>(null);
  // allowed moves per item, loaded from the API on drag or when the card menu opens
  const [movesById, setMovesById] = React.useState<Record<string, ItemStatus[]>>({});
  const [over, setOver] = React.useState<ItemStatus | null>(null);
  const [revision, setRevision] = React.useState<{ itemId: string } | null>(null);
  const [note, setNote] = React.useState("");
  const names = React.useMemo(() => new Map(people.map((p) => [p.id, p.name])), [people]);

  const setParam = useSetParam();

  const columns = React.useMemo(() => {
    const m = new Map<ItemStatus, Item[]>(ITEM_STATUSES.map((s) => [s, []]));
    for (const it of items) m.get(it.status)?.push(it);
    for (const list of m.values()) list.sort((a, b) => a.plannedDate.localeCompare(b.plannedDate));
    return m;
  }, [items]);

  const move = (itemId: string, to: ItemStatus, moveNote?: string) =>
    start(async () => {
      const res = await moveItem({ slug, itemId, to, note: moveNote });
      if (res.ok) {
        toast.success(c.moved(c.itemStatus[to]));
        setRevision(null);
        setNote("");
        setMovesById({});
        router.refresh();
      } else toast.error(res.error);
    });

  const onDragStart = (it: Item, e: React.DragEvent) => {
    e.dataTransfer.setData(DND_MIME, it.id);
    e.dataTransfer.effectAllowed = "move";
    setDragging(it.id);
    setAllowed(movesById[it.id] ?? null);
    void loadMoves(it.id).then(setAllowed);
  };

  const loadMoves = async (itemId: string): Promise<ItemStatus[]> => {
    const res = await proxyRequest<ItemDetail>(contentFiles.item(clientId, itemId));
    const moves = res.ok ? (res.data.allowedMoves ?? []) : [];
    setMovesById((m) => ({ ...m, [itemId]: moves }));
    return moves;
  };

  const pick = (itemId: string, to: ItemStatus) => (to === "REVISION" ? setRevision({ itemId }) : move(itemId, to));

  const endDrag = () => {
    setDragging(null);
    setOver(null);
  };

  const onDrop = (status: ItemStatus, e: React.DragEvent) => {
    e.preventDefault();
    const id = e.dataTransfer.getData(DND_MIME);
    const it = items.find((x) => x.id === id);
    endDrag();
    if (!it || it.status === status) return;
    if (allowed && !allowed.includes(status)) {
      toast.error(c.noMoves);
      return;
    }
    pick(it.id, status);
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <OptionSelect
          label={c.allPlans}
          value={planId}
          onChange={(v) => setParam("plan", v || "all")}
          allLabel={c.allPlans}
          options={plans.map((p) => ({ value: p.id, label: `${monthLabel(p.month)} · ${c.planStatus[p.status]}` }))}
          className="w-[230px]"
        />
        <OptionSelect
          label={c.allAssignees}
          value={assigneeId}
          onChange={(v) => setParam("assignee", v)}
          allLabel={c.allAssignees}
          options={people.map((p) => ({ value: p.id, label: p.name }))}
        />
        {canWrite ? <span className="text-xs text-muted-foreground">{c.boardDragHint}</span> : null}
        {pending ? <Loader2 className="size-4 animate-spin text-muted-foreground" /> : null}
      </div>

      <div className="-mx-4 overflow-x-auto px-4 pb-2 scrollbar-thin md:-mx-6 md:px-6">
        <div className="flex min-w-max gap-3">
          {ITEM_STATUSES.map((status) => {
            const list = columns.get(status) ?? [];
            const highlight = dragging && allowed?.includes(status);
            const blocked = dragging && allowed && !allowed.includes(status) && items.find((x) => x.id === dragging)?.status !== status;
            return (
              <section
                key={status}
                aria-label={c.itemStatus[status]}
                onDragOver={(e) => {
                  if (!dragging) return;
                  e.preventDefault();
                  setOver(status);
                }}
                onDragLeave={() => setOver((o) => (o === status ? null : o))}
                onDrop={(e) => onDrop(status, e)}
                className={cn(
                  "flex w-64 shrink-0 flex-col rounded-2xl bg-muted/40 p-2 transition-colors",
                  highlight && "bg-brand/8 ring-1 ring-brand/30",
                  highlight && over === status && "bg-brand/15",
                  blocked && "opacity-50",
                )}
              >
                <header className="flex items-center gap-2 px-1.5 pb-2 pt-1">
                  <span aria-hidden className="size-2 rounded-full" style={{ background: TONE_VAR[ITEM_TONE[status]] }} />
                  <h2 className="text-[13px] font-semibold">{c.itemStatus[status]}</h2>
                  <span className="tabular ml-auto text-xs text-muted-foreground">{list.length}</span>
                </header>
                <ul className="flex min-h-20 flex-col gap-2">
                  {list.length === 0 ? (
                    <li className="rounded-xl border border-dashed px-3 py-4 text-center text-xs text-muted-foreground">{c.emptyColumn}</li>
                  ) : (
                    list.map((it) => {
                      const due = currentDue(it);
                      const late = isOverdue(it, today);
                      return (
                        <li
                          key={it.id}
                          draggable={canWrite}
                          onDragStart={(e) => onDragStart(it, e)}
                          onDragEnd={endDrag}
                          className={cn(
                            "rounded-xl bg-card p-3 shadow-(--card-shadow) transition-opacity",
                            canWrite && "cursor-grab active:cursor-grabbing",
                            dragging === it.id && "opacity-60",
                          )}
                        >
                          {canWrite ? (
                            <DropdownMenu onOpenChange={(open) => open && !movesById[it.id] && void loadMoves(it.id)}>
                              <DropdownMenuTrigger asChild>
                                <Button size="icon-xs" variant="ghost" className="float-right -mt-1 -mr-1" aria-label={`${c.moveTo}: ${it.title}`}>
                                  <MoreHorizontal />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                <DropdownMenuLabel>{c.moveTo}</DropdownMenuLabel>
                                {!movesById[it.id] ? (
                                  <DropdownMenuItem disabled>{c.loadingMoves}</DropdownMenuItem>
                                ) : movesById[it.id]!.length === 0 ? (
                                  <DropdownMenuItem disabled>{c.noMoves}</DropdownMenuItem>
                                ) : (
                                  movesById[it.id]!.map((to) => (
                                    <DropdownMenuItem key={to} onSelect={() => pick(it.id, to)}>
                                      {c.itemStatus[to]}
                                    </DropdownMenuItem>
                                  ))
                                )}
                              </DropdownMenuContent>
                            </DropdownMenu>
                          ) : null}
                          <Link href={`/clients/${slug}/content/board/${it.id}`} className="block space-y-2 rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring">
                            <p className="line-clamp-2 text-[13px] font-medium leading-snug">{it.title}</p>
                            <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
                              <span className="inline-flex items-center gap-1">
                                <PlatformIcon platform={it.platform} className="size-3.5" />
                                {c.format[it.format]}
                              </span>
                              <span className="tabular">{dayLabel(it.plannedDate)}</span>
                              {it.revisionRound > 0 ? <span>{c.round(it.revisionRound)}</span> : null}
                              {it.hasFinalDesign ? <ImageIcon className="size-3.5 text-[var(--status-good)]" aria-label={c.finalDesign} /> : null}
                            </p>
                            {due ? (
                              <p className={cn("text-xs tabular", late ? "font-medium text-destructive" : "text-muted-foreground")}>
                                {c.stageDue[due.stage]} · {dayLabel(due.date)}
                              </p>
                            ) : null}
                            <p className="truncate text-xs text-muted-foreground">
                              {[names.get(it.copywriterId ?? ""), names.get(it.designerId ?? "")].filter(Boolean).join(" · ") || c.unassigned}
                            </p>
                          </Link>
                        </li>
                      );
                    })
                  )}
                </ul>
              </section>
            );
          })}
        </div>
      </div>

      <Dialog open={!!revision} onOpenChange={(o) => !o && setRevision(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{c.revisionTitle}</DialogTitle>
            <DialogDescription>{c.revisionDesc}</DialogDescription>
          </DialogHeader>
          <Textarea value={note} onChange={(e) => setNote(e.target.value)} placeholder={c.notePlaceholder} rows={3} autoFocus />
          <DialogFooter>
            <Button variant="outline" size="sm" onClick={() => setRevision(null)}>
              {c.cancel}
            </Button>
            <Button size="sm" disabled={pending || !note.trim()} onClick={() => revision && move(revision.itemId, "REVISION", note)}>
              {c.revisionTitle}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
