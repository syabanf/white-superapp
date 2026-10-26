"use client";

import * as React from "react";
import { ChevronDown, ChevronRight, ChevronsDownUp, ChevronsUpDown, Pencil, Plus, Search, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { useErpAction } from "@/features/erp/shared/client";
import { ConfirmAction } from "@/features/erp/shared/confirm-action";
import { ErpEmpty } from "@/features/erp/shared/states";
import { deleteAccountAction } from "../actions";
import type { Account, AccountType } from "../types";
import { fs } from "../strings";
import { AccountDialog } from "./account-dialog";
import { CoaImportDialog } from "./coa-import-dialog";

const a = fs.accounts;

type Node = Account & { children: Node[] };

function buildTree(accounts: Account[]): Node[] {
  const byId = new Map<string, Node>();
  for (const acc of accounts) byId.set(acc.id, { ...acc, children: [] });
  const roots: Node[] = [];
  for (const node of byId.values()) {
    const parent = node.parentId ? byId.get(node.parentId) : undefined;
    if (parent) parent.children.push(node);
    else roots.push(node);
  }
  const sort = (list: Node[]) => {
    list.sort((x, y) => x.code.localeCompare(y.code));
    list.forEach((n) => sort(n.children));
  };
  sort(roots);
  return roots;
}

/** Keeps nodes that match plus their ancestors. */
function filterTree(nodes: Node[], match: (n: Node) => boolean): Node[] {
  return nodes.flatMap((n) => {
    const children = filterTree(n.children, match);
    return match(n) || children.length ? [{ ...n, children }] : [];
  });
}

export function CoaTree({
  accounts,
  accountTypes,
  canWrite,
}: {
  accounts: Account[];
  accountTypes: AccountType[];
  canWrite: boolean;
}) {
  const [query, setQuery] = React.useState("");
  const [showInactive, setShowInactive] = React.useState(false);
  const [collapsed, setCollapsed] = React.useState<Set<string>>(new Set());
  const [dialog, setDialog] = React.useState<{ account?: Account; parentId?: string } | null>(null);
  const { run } = useErpAction();

  const tree = React.useMemo(() => {
    const visible = showInactive ? accounts : accounts.filter((x) => x.isActive);
    const roots = buildTree(visible);
    const q = query.trim().toLowerCase();
    if (!q) return roots;
    return filterTree(roots, (n) => `${n.code} ${n.codeDisplay} ${n.name}`.toLowerCase().includes(q));
  }, [accounts, query, showInactive]);

  const toggle = (id: string) =>
    setCollapsed((s) => {
      const next = new Set(s);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  const parentsIds = accounts.filter((x) => accounts.some((c) => c.parentId === x.id)).map((x) => x.id);

  const rows: React.ReactNode[] = [];
  const walk = (nodes: Node[], depth: number) => {
    for (const n of nodes) {
      const open = query.trim() !== "" || !collapsed.has(n.id);
      rows.push(
        <tr key={n.id} className={cn("border-t hover:bg-accent/40", !n.isActive && "opacity-60")}>
          <td className="py-2 pr-3" style={{ paddingLeft: 12 + depth * 18 }}>
            <span className="flex items-center gap-1.5">
              {n.children.length ? (
                <button
                  type="button"
                  onClick={() => toggle(n.id)}
                  aria-label={open ? a.collapseAll : a.expandAll}
                  className="rounded p-0.5 text-muted-foreground hover:bg-muted"
                >
                  {open ? <ChevronDown className="size-3.5" /> : <ChevronRight className="size-3.5" />}
                </button>
              ) : (
                <span className="w-4.5" />
              )}
              <span className="font-mono text-xs whitespace-nowrap text-muted-foreground">{n.codeDisplay || n.code}</span>
            </span>
          </td>
          <td className={cn("py-2 pr-3", n.level <= 2 && "font-semibold")}>
            {n.name}
            {n.isContra ? <Badge variant="outline" className="ml-2 text-[10px]">{a.contra}</Badge> : null}
            {n.isCashBank ? <Badge variant="secondary" className="ml-2 text-[10px]">{a.cashBank}</Badge> : null}
          </td>
          <td className="py-2 pr-3 text-xs text-muted-foreground">{n.accountTypeName}</td>
          <td className="py-2 pr-3 text-xs">{fs.side[n.normalBalance] ?? "–"}</td>
          <td className="py-2 pr-3 text-xs text-muted-foreground">{fs.cashFlowCategory[n.cashFlowCategory] ?? "–"}</td>
          <td className="py-2 pr-3 text-xs">{n.isPostable ? fs.yes : fs.no}</td>
          <td className="py-2 pr-3">
            {canWrite ? (
              <span className="flex justify-end gap-0.5">
                {n.level < 4 ? (
                  <Button variant="ghost" size="icon-xs" aria-label={a.newChild} onClick={() => setDialog({ parentId: n.id })}>
                    <Plus />
                  </Button>
                ) : null}
                <Button variant="ghost" size="icon-xs" aria-label={fs.edit} onClick={() => setDialog({ account: n })}>
                  <Pencil />
                </Button>
                <ConfirmAction
                  trigger={
                    <Button variant="ghost" size="icon-xs" aria-label={fs.delete} className="text-destructive">
                      <Trash2 />
                    </Button>
                  }
                  title={fs.confirmDeleteTitle}
                  description={a.deleteDesc(n.name)}
                  confirmLabel={fs.delete}
                  destructive
                  onConfirm={async () => (await run(() => deleteAccountAction(n.id), { success: fs.deleted })).ok}
                />
              </span>
            ) : null}
          </td>
        </tr>,
      );
      if (open) walk(n.children, depth + 1);
    }
  };
  walk(tree, 0);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative w-full sm:w-auto">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={a.searchPh}
            aria-label={a.searchPh}
            className="h-8 w-full pl-8 text-[13px] sm:w-64"
          />
        </div>
        <Button variant="ghost" size="sm" onClick={() => setCollapsed(new Set())}>
          <ChevronsUpDown className="size-3.5" /> {a.expandAll}
        </Button>
        <Button variant="ghost" size="sm" onClick={() => setCollapsed(new Set(parentsIds))}>
          <ChevronsDownUp className="size-3.5" /> {a.collapseAll}
        </Button>
        <div className="flex items-center gap-2">
          <Switch id="coa-inactive" checked={showInactive} onCheckedChange={setShowInactive} />
          <Label htmlFor="coa-inactive" className="text-xs font-normal text-muted-foreground">
            {a.showInactive}
          </Label>
        </div>
        {canWrite ? (
          <div className="ml-auto flex gap-2">
            <CoaImportDialog />
            <Button size="sm" onClick={() => setDialog({})}>
              <Plus className="size-4" /> {a.new}
            </Button>
          </div>
        ) : null}
      </div>

      {accounts.length === 0 ? (
        <ErpEmpty title={a.empty} />
      ) : (
        <Card className="gap-0 overflow-hidden p-0">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[860px] text-sm">
              <thead className="bg-muted/40 text-xs text-muted-foreground">
                <tr>
                  <th className="px-3 py-2 text-left font-medium">{fs.code}</th>
                  <th className="px-3 py-2 text-left font-medium">{fs.name}</th>
                  <th className="px-3 py-2 text-left font-medium">{a.accountType}</th>
                  <th className="px-3 py-2 text-left font-medium">{fs.accounts.types.normalBalance}</th>
                  <th className="px-3 py-2 text-left font-medium">{a.cashFlowCategory}</th>
                  <th className="px-3 py-2 text-left font-medium">{a.postable}</th>
                  <th className="w-28" />
                </tr>
              </thead>
              <tbody>{rows}</tbody>
            </table>
          </div>
        </Card>
      )}

      {dialog ? (
        <AccountDialog
          account={dialog.account}
          parentId={dialog.parentId}
          accounts={accounts}
          accountTypes={accountTypes}
          onClose={() => setDialog(null)}
        />
      ) : null}
    </div>
  );
}
