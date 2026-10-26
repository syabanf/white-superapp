"use client";

import * as React from "react";
import { InputGroup, InputGroupAddon, InputGroupInput, InputGroupText } from "@/components/ui/input-group";
import { formatRupiah, parseMoney, parseRupiahInput, toMoneyString } from "./money";

/**
 * Rupiah input. Holds the API decimal string ("1250000.00") as its value and
 * shows thousand separators while typing.
 */
export function MoneyInput({
  value,
  onChange,
  id,
  name,
  placeholder = "0",
  disabled,
  invalid,
  className,
  allowNegative,
}: {
  value: string;
  onChange: (value: string) => void;
  id?: string;
  name?: string;
  placeholder?: string;
  disabled?: boolean;
  invalid?: boolean;
  className?: string;
  allowNegative?: boolean;
}) {
  const display = value === "" ? "" : formatRupiah(value, { symbol: false });
  return (
    <InputGroup className={className} data-disabled={disabled || undefined}>
      <InputGroupAddon>
        <InputGroupText>Rp</InputGroupText>
      </InputGroupAddon>
      <InputGroupInput
        id={id}
        name={name}
        inputMode="numeric"
        autoComplete="off"
        placeholder={placeholder}
        disabled={disabled}
        aria-invalid={invalid || undefined}
        className="text-right tabular"
        value={display}
        onChange={(e) => {
          const raw = e.target.value;
          if (raw.trim() === "") return onChange("");
          let n = parseRupiahInput(raw);
          if (!allowNegative) n = Math.abs(n);
          onChange(toMoneyString(n));
        }}
        onBlur={() => {
          if (value !== "") onChange(toMoneyString(parseMoney(value)));
        }}
      />
    </InputGroup>
  );
}
