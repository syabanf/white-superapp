/**
 * Runs once per server instance, before requests are served. Copies the
 * credentials saved through the setup wizard into process.env so provider
 * adapters (which only read env) pick them up. Failure here must never block
 * boot — without a database the app simply stays in demo mode.
 */
export async function register() {
  if (process.env.NEXT_RUNTIME !== "nodejs") return;
  try {
    const { hydrateIntegrationEnv } = await import("@/lib/integrations");
    await hydrateIntegrationEnv();
  } catch (e) {
    console.warn("[setup] integration env not hydrated:", e instanceof Error ? e.message : e);
  }
}
