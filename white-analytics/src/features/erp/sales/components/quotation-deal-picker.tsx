"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Label } from "@/components/ui/label";
import { SearchableSelect } from "@/features/erp/shared/searchable-select";
import { ss } from "../strings";
import type { Deal } from "../types";

const d = ss.doc;

/**
 * Panel above a new quotation: pick the customer, then pull the deal the
 * quotation belongs to. Choosing a deal reloads the page with `?deal=`.
 */
export function QuotationDealPicker({ deals, dealId }: { deals: Deal[]; dealId?: string }) {
  const router = useRouter();
  const current = deals.find((x) => x.id === dealId);
  const [client, setClient] = React.useState(current?.clientName ?? "");
  const clients = [...new Set(deals.map((x) => x.clientName).filter(Boolean))].sort();
  const forClient = client ? deals.filter((x) => x.clientName === client) : deals;

  return (
    <div className="grid gap-(--gap-inner) rounded-[1.5rem] border border-brand/20 bg-brand/5 p-(--card-pad) md:grid-cols-2" data-print-hide>
      <div className="flex min-w-0 flex-col gap-1.5">
        <Label htmlFor="q-client">{d.customer}</Label>
        <SearchableSelect
          id="q-client"
          value={client}
          onChange={setClient}
          options={clients.map((c) => ({ value: c, label: c }))}
          placeholder={d.customerPlaceholder}
          clearable
        />
      </div>
      <div className="flex min-w-0 flex-col gap-1.5">
        <Label htmlFor="q-deal">{d.fromDeal}</Label>
        <SearchableSelect
          id="q-deal"
          value={dealId ?? ""}
          onChange={(id) => id && router.replace(`/erp/sales/quotations/new?deal=${id}`)}
          options={forClient.map((x) => ({ value: x.id, label: x.title, hint: client ? undefined : x.clientName }))}
          placeholder={d.fromDealPlaceholder}
          emptyText={d.noDeals}
        />
      </div>
    </div>
  );
}
