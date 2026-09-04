import { defineConfig } from "vitest/config";
import path from "node:path";
import { fileURLToPath } from "node:url";
const __dirname = path.dirname(fileURLToPath(import.meta.url));

/** DB-backed smoke tests (need DATABASE_URL). Run: pnpm sync:smoke */
export default defineConfig({
  test: {
    include: ["scripts/**/*.smoke.test.ts"],
    environment: "node",
    setupFiles: ["dotenv/config"],
    testTimeout: 120_000,
    hookTimeout: 120_000,
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
      "server-only": path.resolve(__dirname, "tests/stubs/server-only.ts"),
    },
  },
});
