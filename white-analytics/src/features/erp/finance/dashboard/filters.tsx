"use client";

import { DashboardFilterBar } from "@/features/erp/shared/dashboard-filter-bar";
import type { ResolvedPeriod } from "@/features/erp/shared/dashboard-period";
import { SelectFilter, type FilterOption } from "@/features/erp/shared/url-filters";
import { ds } from "./strings";

export const DIMENSION_PARAMS = ["accountType", "accountGroup", "department", "cashAccount"];

const TYPE_OPTIONS = Object.entries(ds.types).map(([value, label]) => ({ value, label }));

/** Period, comparison and the dimension filters the current tab uses. */
export function FinanceFilters({
  period,
  tab,
  groups,
  cashAccounts,
}: {
  period: ResolvedPeriod;
  tab: string;
  groups: FilterOption[];
  cashAccounts: FilterOption[];
}) {
  const accounts = tab === "profit" || tab === "spend";
  return (
    <DashboardFilterBar period={period} params={DIMENSION_PARAMS}>
      {accounts ? (
        <>
          <SelectFilter param="accountType" label={ds.filters.accountType} allLabel={ds.filters.allTypes} options={TYPE_OPTIONS} />
          <SelectFilter param="accountGroup" label={ds.filters.accountGroup} allLabel={ds.filters.allGroups} options={groups} className="max-w-60" />
        </>
      ) : null}
      {tab === "cash" ? (
        <SelectFilter param="cashAccount" label={ds.filters.cashAccount} allLabel={ds.filters.allCash} options={cashAccounts} />
      ) : null}
    </DashboardFilterBar>
  );
}
