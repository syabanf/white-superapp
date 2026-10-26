"use client";

import * as React from "react";
import { Loader2, Plus, Sparkles, Trash2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { formatRupiah, parseMoney, toMoneyString } from "@/features/erp/shared/money";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { cn } from "@/lib/utils";
import { createQuotationAction, updateQuotationAction } from "../actions";
import { sumsTo100 } from "../lib";
import { TIER, ss } from "../strings";
import type { CatalogItem, Quotation, QuotationInput, RetainerDiscount } from "../types";
import { Fld, SimpleSelect } from "./ui";

const s = ss.quotations;

type Item = QuotationInput["items"][number] & { key: string };
type Term = QuotationInput["terms"][number] & { key: string };

const uid = () => Math.random().toString(36).slice(2);

function initial(q: Quotation | null): { items: Item[]; terms: Term[] } {
  if (!q) return { items: [], terms: [{ key: uid(), label: "Pembayaran penuh", percent: 100, dueDate: "" }] };
  return {
    items: q.items.map((it) => ({
      key: it.id || uid(),
      itemType: it.itemType,
      catalogId: it.catalogId,
      description: it.description,
      scope: it.scope,
      unit: it.unit,
      qty: it.qty,
      unitPrice: it.unitPrice,
    })),
    terms: q.terms.map((t) => ({ key: t.id || uid(), label: t.label, percent: t.percent, dueDate: t.dueDate })),
  };
}

/** Best retainer discount for the longest retainer line. */
function retainerDiscountFor(months: number, rules: RetainerDiscount[]): number {
  return rules.filter((r) => months >= r.minMonths).reduce((best, r) => Math.max(best, r.discountPercent), 0);
}

/**
 * Quotation builder: catalog items with package tiers, free items, discount,
 * PPN and payment terms. Totals preview here; the server computes the final
 * numbers.
 */
export function QuotationEditor({
  quotation,
  dealId,
  catalog,
  retainerRules,
}: {
  quotation: Quotation | null;
  dealId: string;
  catalog: CatalogItem[];
  retainerRules: RetainerDiscount[];
}) {
  const router = useRouter();
  const init = React.useMemo(() => initial(quotation), [quotation]);
  const [items, setItems] = React.useState<Item[]>(init.items);
  const [terms, setTerms] = React.useState<Term[]>(init.terms);
  const [discount, setDiscount] = React.useState(quotation?.discountPercent ?? 0);
  const [usePpn, setUsePpn] = React.useState(quotation?.usePpn ?? true);
  const [ppn, setPpn] = React.useState(quotation?.ppnPercent || 11);
  const [notes, setNotes] = React.useState(quotation?.notes ?? "");
  const [validUntil, setValidUntil] = React.useState(quotation?.validUntil ?? "");
  const [error, setError] = React.useState<string | null>(null);
  const { pending, run } = useErpAction();

  const catalogById = Object.fromEntries(catalog.map((c) => [c.id, c]));
  const subtotal = items.reduce((acc, it) => acc + (Number(it.qty) || 0) * parseMoney(it.unitPrice), 0);
  const discNominal = Math.round(subtotal * (Number(discount) || 0)) / 100;
  const ppnNominal = usePpn ? Math.round((subtotal - discNominal) * (Number(ppn) || 0)) / 100 : 0;
  const total = subtotal - discNominal + ppnNominal;
  const termsTotal = terms.reduce((a, t) => a + (Number(t.percent) || 0), 0);
  const retainerMonths = items
    .filter((it) => it.catalogId && catalogById[it.catalogId]?.isRetainer)
    .reduce((m, it) => Math.max(m, Number(it.qty) || 0), 0);
  const suggested = retainerDiscountFor(retainerMonths, retainerRules);

  const patchItem = (key: string, p: Partial<Item>) => setItems((list) => list.map((it) => (it.key === key ? { ...it, ...p } : it)));
  const patchTerm = (key: string, p: Partial<Term>) => setTerms((list) => list.map((t) => (t.key === key ? { ...t, ...p } : t)));

  const addCatalog = (id: string, tier: string) => {
    const c = catalogById[id];
    if (!c) return;
    const t = c.tiers.find((x) => x.tier === tier);
    setItems((list) => [
      ...list,
      {
        key: uid(),
        itemType: "katalog",
        catalogId: c.id,
        description: t ? `${c.name} (${TIER[t.tier] ?? t.tier})` : c.name,
        scope: [c.description, t?.description].filter(Boolean).join("\n"),
        unit: c.unit,
        qty: c.isRetainer ? 3 : 1,
        unitPrice: t?.price ?? c.defaultPrice,
      },
    ]);
  };

  const save = () => {
    setError(null);
    if (!items.length) return setError(s.noItems);
    if (!sumsTo100(terms.map((t) => t.percent))) return setError(s.termsInvalid);
    const input: QuotationInput = {
      usePpn,
      ppnPercent: Number(ppn) || 0,
      discountPercent: Number(discount) || 0,
      notes,
      validUntil,
      items: items.map((it) => ({
        itemType: it.itemType,
        catalogId: it.catalogId,
        description: it.description,
        scope: it.scope,
        unit: it.unit,
        qty: Number(it.qty) || 0,
        unitPrice: toMoneyString(parseMoney(it.unitPrice)),
      })),
      terms: terms.map((t) => ({ label: t.label, percent: Number(t.percent) || 0, dueDate: t.dueDate })),
    };
    void run(() => (quotation ? updateQuotationAction(quotation.id, input) : createQuotationAction(dealId, input)), {
      success: ss.saved,
      onOk: (q) => {
        if (!quotation) router.push(`/erp/sales/quotations/${q.id}`);
      },
      onFail: (r) => setError(r.error),
    });
  };

  return (
    <div className="flex flex-col gap-4">
      <Card className="gap-0 py-0">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b px-5 py-3">
          <h2 className="text-[15px] font-semibold">{s.items}</h2>
          <div className="flex flex-wrap gap-2">
            <CatalogAdder catalog={catalog} onAdd={addCatalog} />
            <Button
              size="sm"
              variant="outline"
              onClick={() =>
                setItems((l) => [...l, { key: uid(), itemType: "bebas", catalogId: "", description: "", scope: "", unit: "paket", qty: 1, unitPrice: "" }])
              }
            >
              <Plus className="size-3.5" /> {s.addFree}
            </Button>
          </div>
        </div>
        {items.length === 0 ? <p className="px-5 py-6 text-sm text-muted-foreground">{s.noItems}</p> : null}
        <ul className="divide-y">
          {items.map((it) => (
            <li key={it.key} className="grid gap-3 px-5 py-4 md:grid-cols-12">
              <div className="flex flex-col gap-2 md:col-span-6">
                <Input
                  aria-label={s.description}
                  placeholder={s.description}
                  value={it.description}
                  onChange={(e) => patchItem(it.key, { description: e.target.value })}
                />
                <Textarea
                  aria-label={s.scope}
                  placeholder={s.scope}
                  rows={2}
                  value={it.scope}
                  onChange={(e) => patchItem(it.key, { scope: e.target.value })}
                  className="text-xs"
                />
              </div>
              <div className="grid grid-cols-3 gap-2 md:col-span-5 md:grid-cols-[70px_80px_1fr]">
                <Fld label={s.qty}>
                  <Input
                    type="number"
                    min={0}
                    step="any"
                    value={it.qty}
                    onChange={(e) => patchItem(it.key, { qty: Number(e.target.value) })}
                    className="tabular"
                  />
                </Fld>
                <Fld label={s.unit}>
                  <Input value={it.unit} onChange={(e) => patchItem(it.key, { unit: e.target.value })} />
                </Fld>
                <Fld label={s.price}>
                  <MoneyInput value={it.unitPrice} onChange={(v) => patchItem(it.key, { unitPrice: v })} />
                </Fld>
                <p className="col-span-3 text-right text-xs text-muted-foreground">
                  {s.lineTotal}{" "}
                  <span className="font-medium text-foreground tabular">
                    {formatRupiah((Number(it.qty) || 0) * parseMoney(it.unitPrice))}
                  </span>
                </p>
              </div>
              <div className="flex justify-end md:col-span-1">
                <Button
                  size="icon-sm"
                  variant="ghost"
                  aria-label={ss.delete}
                  onClick={() => setItems((l) => l.filter((x) => x.key !== it.key))}
                >
                  <Trash2 className="size-3.5" />
                </Button>
              </div>
            </li>
          ))}
        </ul>
      </Card>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card className="gap-4 px-5 py-4">
          <div className="flex items-center justify-between">
            <h2 className="text-[15px] font-semibold">{s.termsTitle}</h2>
            <Button
              size="sm"
              variant="outline"
              onClick={() => setTerms((l) => [...l, { key: uid(), label: `Termin ${l.length + 1}`, percent: 0, dueDate: "" }])}
            >
              <Plus className="size-3.5" /> {s.addTerm}
            </Button>
          </div>
          <ul className="flex flex-col gap-2">
            {terms.map((t) => (
              <li key={t.key} className="grid grid-cols-[1fr_72px_150px_auto] items-center gap-2">
                <Input aria-label={s.termLabel} value={t.label} onChange={(e) => patchTerm(t.key, { label: e.target.value })} />
                <Input
                  aria-label={s.termPercent}
                  type="number"
                  min={0}
                  max={100}
                  step="any"
                  value={t.percent}
                  onChange={(e) => patchTerm(t.key, { percent: Number(e.target.value) })}
                  className="tabular"
                />
                <DatePicker value={t.dueDate} onChange={(v) => patchTerm(t.key, { dueDate: v })} placeholder={s.termDue} clearable />
                <Button
                  size="icon-sm"
                  variant="ghost"
                  aria-label={ss.delete}
                  onClick={() => setTerms((l) => l.filter((x) => x.key !== t.key))}
                  disabled={terms.length === 1}
                >
                  <Trash2 className="size-3.5" />
                </Button>
              </li>
            ))}
          </ul>
          <p className={cn("text-xs tabular", sumsTo100(terms.map((t) => t.percent)) ? "text-muted-foreground" : "text-destructive")}>
            {s.termsTotal(Math.round(termsTotal * 100) / 100)}
          </p>
          <Fld label={s.validUntil}>
            <DatePicker value={validUntil} onChange={setValidUntil} clearable />
          </Fld>
          <Fld label={s.notes}>
            <Textarea rows={4} value={notes} onChange={(e) => setNotes(e.target.value)} />
          </Fld>
        </Card>

        <Card className="gap-3 px-5 py-4">
          <Row label={s.subtotal} value={formatRupiah(subtotal)} />
          <div className="flex items-center justify-between gap-3">
            <label htmlFor="q-discount" className="text-sm">
              {s.discount}
            </label>
            <Input
              id="q-discount"
              type="number"
              min={0}
              max={100}
              step="any"
              value={discount}
              onChange={(e) => setDiscount(Number(e.target.value))}
              className="w-24 text-right tabular"
            />
          </div>
          {suggested > 0 && suggested !== discount ? (
            <button
              type="button"
              onClick={() => setDiscount(suggested)}
              className="self-end text-xs text-brand underline-offset-2 hover:underline"
            >
              {s.retainerHint(retainerMonths, suggested)}
            </button>
          ) : null}
          <Row label={s.discountNominal} value={`-${formatRupiah(discNominal)}`} muted />
          <div className="flex items-center justify-between gap-3">
            <label className="flex items-center gap-2 text-sm">
              <Switch checked={usePpn} onCheckedChange={setUsePpn} /> {s.usePpn}
            </label>
            {usePpn ? (
              <Input
                aria-label={s.ppn}
                type="number"
                min={0}
                step="any"
                value={ppn}
                onChange={(e) => setPpn(Number(e.target.value))}
                className="w-24 text-right tabular"
              />
            ) : null}
          </div>
          {usePpn ? <Row label={s.ppnNominal} value={formatRupiah(ppnNominal)} muted /> : null}
          <div className="border-t pt-3">
            <Row label={s.total} value={formatRupiah(total)} strong />
          </div>
          {error ? <p className="text-sm text-destructive">{error}</p> : null}
          <Button onClick={save} disabled={pending} className="mt-2">
            {pending ? <Loader2 className="size-4 animate-spin" /> : null}
            {ss.save}
          </Button>
        </Card>
      </div>
    </div>
  );
}

function Row({ label, value, muted, strong }: { label: string; value: string; muted?: boolean; strong?: boolean }) {
  return (
    <div className={cn("flex items-center justify-between gap-3 text-sm", muted && "text-muted-foreground", strong && "text-base font-semibold")}>
      <span>{label}</span>
      <span className="tabular">{value}</span>
    </div>
  );
}

function CatalogAdder({ catalog, onAdd }: { catalog: CatalogItem[]; onAdd: (id: string, tier: string) => void }) {
  const [id, setId] = React.useState("");
  const [tier, setTier] = React.useState("standard");
  const item = catalog.find((c) => c.id === id);
  return (
    <div className="flex flex-wrap items-center gap-2">
      <SimpleSelect
        size="sm"
        value={id}
        onChange={setId}
        placeholder={s.addCatalog}
        options={catalog.filter((c) => c.isActive).map((c) => ({ value: c.id, label: `${c.name} · ${formatRupiah(c.defaultPrice)}` }))}
        className="w-64"
      />
      {item?.tiers.length ? (
        <SimpleSelect
          size="sm"
          value={tier}
          onChange={setTier}
          options={item.tiers.map((t) => ({ value: t.tier, label: `${TIER[t.tier] ?? t.tier} · ${formatRupiah(t.price)}` }))}
          className="w-48"
        />
      ) : null}
      <Button
        size="sm"
        variant="outline"
        disabled={!id}
        onClick={() => {
          onAdd(id, tier);
          setId("");
        }}
      >
        <Sparkles className="size-3.5" /> {ss.add}
      </Button>
    </div>
  );
}
