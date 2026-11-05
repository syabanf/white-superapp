"use client";

import { SimpleSelect } from "@/features/erp/sales/components/ui";
import { fs } from "../strings";
import type { ProjectTag } from "../types";

/** Active tags, plus the current one when it has since been deactivated. */
export function TagPicker({ id, tags, value, onChange }: { id?: string; tags: ProjectTag[]; value: string; onChange: (v: string) => void }) {
  const options = tags
    .filter((t) => t.isActive || t.id === value)
    .map((t) => ({ value: t.id, label: t.clientName ? `${t.name} · ${t.clientName}` : t.name }));
  return <SimpleSelect id={id} value={value || "-"} onChange={(x) => onChange(x === "-" ? "" : x)} options={[{ value: "-", label: fs.projects.noTag }, ...options]} />;
}
