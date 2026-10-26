"use client";

import { SearchableSelect } from "@/features/erp/shared/searchable-select";
import type { PostableAccount } from "../types";
import { fs } from "../strings";

/** Searchable picker over postable accounts ("1 1 01 001 Kas"). */
export function AccountPicker({
  accounts,
  value,
  onChange,
  id,
  placeholder = fs.pickAccount,
  invalid,
  disabled,
  clearable,
  filter,
}: {
  accounts: PostableAccount[];
  value: string;
  onChange: (id: string) => void;
  id?: string;
  placeholder?: string;
  invalid?: boolean;
  disabled?: boolean;
  clearable?: boolean;
  filter?: (a: PostableAccount) => boolean;
}) {
  const options = (filter ? accounts.filter(filter) : accounts).map((a) => ({
    value: a.id,
    label: a.name,
    hint: a.codeDisplay || a.code,
  }));
  return (
    <SearchableSelect
      id={id}
      options={options}
      value={value}
      onChange={onChange}
      placeholder={placeholder}
      searchPlaceholder={fs.searchAccount}
      emptyText={fs.noAccount}
      invalid={invalid}
      disabled={disabled}
      clearable={clearable}
    />
  );
}
