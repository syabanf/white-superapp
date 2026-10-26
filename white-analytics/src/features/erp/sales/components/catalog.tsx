"use client";

import * as React from "react";
import { Loader2, Pencil, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { useErpAction } from "@/features/erp/shared/client";
import { deleteApprovalRuleAction, saveApprovalRuleAction, saveCatalogItemAction, saveRetainerDiscountsAction } from "../actions";
import { nameOf } from "../lib";
import { SERVICE_LINE, TIER, lbl, opts, ss } from "../strings";
import type { ApprovalRule, CatalogItem, RetainerDiscount } from "../types";
import { Panel } from "./blocks";
import { ConfirmAction, Fld, FormDialog, FormFooter, type Option, Picker, SalesBadge, SimpleSelect } from "./ui";

const c = ss.catalog;
const a = ss.approvals;

const EMPTY_ITEM: CatalogItem = {
  id: "",
  code: "",
  name: "",
  serviceLine: "social_media_management",
  unit: "bulan",
  defaultPrice: "",
  description: "",
  isRetainer: true,
  isDefaultForLine: false,
  isActive: true,
  sortOrder: 0,
  tiers: [],
};

/** Catalog with package tiers, retainer discounts and approval rules. Editable for ADMIN. */
export function CatalogSettings({
  catalog,
  retainer,
  rules,
  owners,
  admin,
}: {
  catalog: CatalogItem[];
  retainer: RetainerDiscount[];
  rules: ApprovalRule[];
  owners: Option[];
  admin: boolean;
}) {
  const names = Object.fromEntries(owners.map((o) => [o.value, o.label]));
  return (
    <div className="flex flex-col gap-4">
      <Panel
        title={ss.quotations.tabs.catalog}
        flush
        actions={
          admin ? (
            <CatalogItemDialog
              item={null}
              trigger={
                <Button size="sm">
                  <Plus className="size-3.5" /> {c.add}
                </Button>
              }
            />
          ) : null
        }
      >
        <ul className="divide-y">
          {[...catalog].sort((x, y) => x.sortOrder - y.sortOrder).map((it) => (
            <li key={it.id} className="flex flex-wrap items-start gap-3 px-5 py-3">
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-medium">{it.name}</span>
                  <span className="text-xs text-muted-foreground">{it.code}</span>
                  {!it.isActive ? <SalesBadge value="ended" map={{ ended: ss.inactive }} /> : null}
                </div>
                <p className="text-xs text-muted-foreground">
                  {lbl(SERVICE_LINE, it.serviceLine)} · {it.unit}
                  {it.isRetainer ? ` · ${c.retainer}` : ""}
                  {it.isDefaultForLine ? ` · ${c.defaultForLine}` : ""}
                </p>
                {it.tiers.length ? (
                  <div className="mt-1.5 flex flex-wrap gap-1.5">
                    {it.tiers.map((t) => (
                      <span key={t.tier} className="rounded-full border px-2 py-0.5 text-xs" title={t.description}>
                        {lbl(TIER, t.tier)} · <Money value={t.price} />
                      </span>
                    ))}
                  </div>
                ) : null}
              </div>
              <Money value={it.defaultPrice} className="font-medium" />
              {admin ? (
                <CatalogItemDialog
                  item={it}
                  trigger={
                    <Button size="icon-sm" variant="ghost" aria-label={ss.edit}>
                      <Pencil className="size-3.5" />
                    </Button>
                  }
                />
              ) : null}
            </li>
          ))}
        </ul>
      </Panel>
      <div className="grid gap-4 lg:grid-cols-2">
        <Panel title={c.retainerDiscounts}>
          <RetainerEditor rules={retainer} admin={admin} />
        </Panel>
        <Panel
          title={a.rules}
          flush
          actions={
            admin ? (
              <ApprovalRuleDialog
                rule={null}
                owners={owners}
                trigger={
                  <Button size="sm" variant="outline">
                    <Plus className="size-3.5" /> {a.addRule}
                  </Button>
                }
              />
            ) : null
          }
        >
          <ul className="divide-y">
            {[...rules].sort((x, y) => x.level - y.level).map((r) => (
              <li key={r.id} className="flex items-center gap-3 px-5 py-3 text-sm">
                <span className="min-w-0 flex-1">
                  <span className="block font-medium">{r.name}</span>
                  <span className="text-xs text-muted-foreground">
                    {a.level(r.level)} · ≥ {r.minDiscountPercent}% ·{" "}
                    {r.approverUserId ? nameOf(names, r.approverUserId) : r.approverRole}
                    {!r.isActive ? ` · ${ss.inactive}` : ""}
                  </span>
                </span>
                {admin ? (
                  <>
                    <ApprovalRuleDialog
                      rule={r}
                      owners={owners}
                      trigger={
                        <Button size="icon-sm" variant="ghost" aria-label={ss.edit}>
                          <Pencil className="size-3.5" />
                        </Button>
                      }
                    />
                    <ConfirmAction
                      title={ss.confirmDelete}
                      action={() => deleteApprovalRuleAction(r.id)}
                      trigger={
                        <Button size="icon-sm" variant="ghost" aria-label={ss.delete}>
                          <Trash2 className="size-3.5" />
                        </Button>
                      }
                    />
                  </>
                ) : null}
              </li>
            ))}
          </ul>
        </Panel>
      </div>
      {!admin ? <p className="text-xs text-muted-foreground">{ss.adminOnly}</p> : null}
    </div>
  );
}

function CatalogItemDialog({ item, trigger }: { item: CatalogItem | null; trigger: React.ReactNode }) {
  return (
    <FormDialog trigger={trigger} title={c.addTitle} size="lg">
      {(close) => <CatalogItemForm item={item ?? EMPTY_ITEM} close={close} />}
    </FormDialog>
  );
}

function CatalogItemForm({ item, close }: { item: CatalogItem; close: () => void }) {
  const [v, setV] = React.useState<CatalogItem>(item);
  const { pending, run } = useErpAction();
  const set = <K extends keyof CatalogItem>(k: K, val: CatalogItem[K]) => setV((p) => ({ ...p, [k]: val }));
  const tierFor = (tier: string) => v.tiers.find((t) => t.tier === tier);
  const setTier = (tier: string, patch: { price?: string; description?: string } | null) =>
    setV((p) => {
      const rest = p.tiers.filter((t) => t.tier !== tier);
      if (!patch) return { ...p, tiers: rest };
      const cur = p.tiers.find((t) => t.tier === tier) ?? { tier, price: "", description: "" };
      return { ...p, tiers: [...rest, { ...cur, ...patch }].sort((x, y) => Object.keys(TIER).indexOf(x.tier) - Object.keys(TIER).indexOf(y.tier)) };
    });
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveCatalogItemAction(v), { success: ss.saved, onOk: close });
      }}
    >
      <Fld label={`${c.item} *`} htmlFor="cat-name">
        <Input id="cat-name" value={v.name} onChange={(e) => set("name", e.target.value)} />
      </Fld>
      <Fld label={`${c.code} *`} htmlFor="cat-code">
        <Input id="cat-code" value={v.code} onChange={(e) => set("code", e.target.value)} />
      </Fld>
      <Fld label={c.serviceLine} htmlFor="cat-line">
        <SimpleSelect id="cat-line" value={v.serviceLine} onChange={(x) => set("serviceLine", x)} options={opts(SERVICE_LINE)} />
      </Fld>
      <Fld label={ss.quotations.unit} htmlFor="cat-unit">
        <Input id="cat-unit" value={v.unit} onChange={(e) => set("unit", e.target.value)} />
      </Fld>
      <Fld label={c.defaultPrice} htmlFor="cat-price">
        <MoneyInput id="cat-price" value={v.defaultPrice} onChange={(x) => set("defaultPrice", x)} />
      </Fld>
      <Fld label={ss.automation.order} htmlFor="cat-order">
        <Input id="cat-order" type="number" value={v.sortOrder} onChange={(e) => set("sortOrder", Number(e.target.value))} />
      </Fld>
      <Fld label={ss.quotations.description} htmlFor="cat-desc" className="sm:col-span-2">
        <Textarea id="cat-desc" rows={2} value={v.description} onChange={(e) => set("description", e.target.value)} />
      </Fld>
      <div className="flex flex-wrap gap-4 text-sm sm:col-span-2">
        <label className="flex items-center gap-2">
          <Checkbox checked={v.isRetainer} onCheckedChange={(x) => set("isRetainer", x === true)} /> {c.retainer}
        </label>
        <label className="flex items-center gap-2">
          <Checkbox checked={v.isDefaultForLine} onCheckedChange={(x) => set("isDefaultForLine", x === true)} /> {c.defaultForLine}
        </label>
        <label className="flex items-center gap-2">
          <Checkbox checked={v.isActive} onCheckedChange={(x) => set("isActive", x === true)} /> {ss.active}
        </label>
      </div>
      <fieldset className="flex flex-col gap-2 sm:col-span-2">
        <legend className="mb-1 text-sm font-medium">{c.tiers}</legend>
        {Object.entries(TIER).map(([tier, label]) => {
          const t = tierFor(tier);
          return (
            <div key={tier} className="grid grid-cols-[100px_160px_1fr] items-center gap-2">
              <label className="flex items-center gap-2 text-sm">
                <Checkbox checked={!!t} onCheckedChange={(x) => setTier(tier, x === true ? {} : null)} /> {label}
              </label>
              <MoneyInput value={t?.price ?? ""} onChange={(x) => setTier(tier, { price: x })} disabled={!t} />
              <Input
                aria-label={`${label} ${ss.quotations.description}`}
                value={t?.description ?? ""}
                onChange={(e) => setTier(tier, { description: e.target.value })}
                disabled={!t}
              />
            </div>
          );
        })}
      </fieldset>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

function RetainerEditor({ rules, admin }: { rules: RetainerDiscount[]; admin: boolean }) {
  const [rows, setRows] = React.useState(rules);
  const { pending, run } = useErpAction();
  return (
    <div className="flex flex-col gap-2">
      <div className="grid grid-cols-[1fr_1fr_auto] gap-2 text-xs text-muted-foreground">
        <span>{c.minMonths}</span>
        <span>{c.discountPercent}</span>
        <span className="w-8" />
      </div>
      {rows.map((r, i) => (
        <div key={i} className="grid grid-cols-[1fr_1fr_auto] gap-2">
          <Input
            type="number"
            min={1}
            value={r.minMonths}
            disabled={!admin}
            aria-label={c.minMonths}
            onChange={(e) => setRows((l) => l.map((x, j) => (j === i ? { ...x, minMonths: Number(e.target.value) } : x)))}
          />
          <Input
            type="number"
            min={0}
            max={100}
            step="any"
            value={r.discountPercent}
            disabled={!admin}
            aria-label={c.discountPercent}
            onChange={(e) => setRows((l) => l.map((x, j) => (j === i ? { ...x, discountPercent: Number(e.target.value) } : x)))}
          />
          <Button
            size="icon-sm"
            variant="ghost"
            aria-label={ss.delete}
            disabled={!admin}
            onClick={() => setRows((l) => l.filter((_, j) => j !== i))}
          >
            <Trash2 className="size-3.5" />
          </Button>
        </div>
      ))}
      {admin ? (
        <div className="mt-2 flex justify-between">
          <Button size="sm" variant="outline" onClick={() => setRows((l) => [...l, { minMonths: 12, discountPercent: 0 }])}>
            <Plus className="size-3.5" /> {c.addRule}
          </Button>
          <Button size="sm" disabled={pending} onClick={() => void run(() => saveRetainerDiscountsAction(rows), { success: ss.saved })}>
            {pending ? <Loader2 className="size-3.5 animate-spin" /> : null}
            {ss.save}
          </Button>
        </div>
      ) : null}
    </div>
  );
}

function ApprovalRuleDialog({ rule, owners, trigger }: { rule: ApprovalRule | null; owners: Option[]; trigger: React.ReactNode }) {
  return (
    <FormDialog trigger={trigger} title={a.rules}>
      {(close) => <ApprovalRuleForm rule={rule} owners={owners} close={close} />}
    </FormDialog>
  );
}

function ApprovalRuleForm({ rule, owners, close }: { rule: ApprovalRule | null; owners: Option[]; close: () => void }) {
  const [v, setV] = React.useState<ApprovalRule>(
    rule ?? { id: "", name: "", level: 1, minDiscountPercent: 10, approverRole: "sales.approve", approverUserId: "", isActive: true },
  );
  const { pending, run } = useErpAction();
  const set = <K extends keyof ApprovalRule>(k: K, val: ApprovalRule[K]) => setV((p) => ({ ...p, [k]: val }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveApprovalRuleAction(v), { success: ss.saved, onOk: close });
      }}
    >
      <Fld label={`${a.ruleName} *`} htmlFor="ar-name" className="sm:col-span-2">
        <Input id="ar-name" value={v.name} onChange={(e) => set("name", e.target.value)} />
      </Fld>
      <Fld label="Level" htmlFor="ar-level">
        <Input id="ar-level" type="number" min={1} value={v.level} onChange={(e) => set("level", Number(e.target.value))} />
      </Fld>
      <Fld label={a.minDiscount} htmlFor="ar-min">
        <Input id="ar-min" type="number" min={0} max={100} step="any" value={v.minDiscountPercent} onChange={(e) => set("minDiscountPercent", Number(e.target.value))} />
      </Fld>
      <Fld label={a.approverRole} htmlFor="ar-role">
        <SimpleSelect
          id="ar-role"
          value={v.approverRole}
          onChange={(x) => set("approverRole", x)}
          options={[
            { value: "sales.approve", label: "sales.approve" },
            { value: "ADMIN", label: "ADMIN" },
          ]}
          emptyLabel={ss.dash}
        />
      </Fld>
      <Fld label={a.approverUser} htmlFor="ar-user">
        <Picker id="ar-user" value={v.approverUserId} onChange={(x) => set("approverUserId", x)} options={owners} allowEmpty={ss.dash} />
      </Fld>
      <label className="flex items-center gap-2 text-sm sm:col-span-2">
        <Checkbox checked={v.isActive} onCheckedChange={(x) => set("isActive", x === true)} /> {ss.active}
      </label>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}
