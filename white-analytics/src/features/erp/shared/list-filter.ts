/** Filters for short master lists the server returns whole. */
export function matchesText(q: string | undefined, ...fields: (string | null | undefined)[]) {
  const needle = q?.trim().toLowerCase();
  return !needle || fields.some((f) => f?.toLowerCase().includes(needle));
}

export function matchesActive(status: string | undefined, isActive: boolean) {
  return !status || (status === "active") === isActive;
}
