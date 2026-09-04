/**
 * Resolve a provider implementation per call instead of at import time, so an
 * adapter switches from mock to real the moment credentials land in
 * process.env (hydrated from the setup wizard) — no server restart needed.
 */
export function lazyProvider<T extends object>(pick: () => T): T {
  return new Proxy({} as T, {
    get(_target, key) {
      const impl = pick();
      const value = Reflect.get(impl, key) as unknown;
      return typeof value === "function" ? (value as (...a: unknown[]) => unknown).bind(impl) : value;
    },
  });
}
