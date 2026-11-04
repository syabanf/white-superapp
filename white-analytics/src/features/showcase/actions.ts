"use server";

import { z } from "zod";
import { fail, ok, type ActionResult } from "@/lib/action-result";
import { aiChatTarget } from "@/lib/providers/ai-chat";
import { getSessionUser } from "@/lib/rbac";
import { t } from "@/i18n/id";
import { buildChatMessages, parseChatReply, type ChatReply } from "./chat";
import { CHAT_SCENARIOS } from "./data";

const TIMEOUT_MS = 60_000;

const inputSchema = z.object({
  scenarioId: z.string(),
  turns: z
    .array(z.object({ from: z.enum(["them", "bot", "me"]), name: z.string().max(40), text: z.string().trim().min(1).max(2000) }))
    .min(1)
    .max(40),
});

/** Answers the text typed in the WhatsApp demo with the configured text AI. Nothing is stored or sent to WhatsApp. */
export async function replyToDemoChat(input: z.input<typeof inputSchema>): Promise<ActionResult<ChatReply>> {
  if (!(await getSessionUser())) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");

  const parsed = inputSchema.safeParse(input);
  const scenario = CHAT_SCENARIOS.find((s) => s.id === parsed.data?.scenarioId);
  if (!parsed.success || !scenario) return fail("Pesan tidak valid. Teks maksimal 2.000 karakter.", "VALIDATION");

  const { label, url, apiKey, model } = aiChatTarget();
  if (!apiKey) return fail("Kunci AI belum diisi. Isi OLLAMA_API_KEY atau OPENROUTER_API_KEY untuk mencoba demo ini.", "PERMISSION");

  try {
    const res = await fetch(url, {
      method: "POST",
      signal: AbortSignal.timeout(TIMEOUT_MS),
      headers: { Authorization: `Bearer ${apiKey}`, "Content-Type": "application/json" },
      body: JSON.stringify({
        model,
        max_tokens: 800,
        temperature: 0.3,
        response_format: { type: "json_object" },
        messages: buildChatMessages(scenario, parsed.data.turns),
      }),
    });
    if (!res.ok) return fail(`${label} mengembalikan status ${res.status}. Coba kirim lagi.`, "UNKNOWN");
    const json = (await res.json()) as { choices?: { message?: { content?: string } }[] };
    const content = json.choices?.[0]?.message?.content;
    if (!content?.trim()) return fail(`${label} tidak mengembalikan balasan. Coba kirim lagi.`, "UNKNOWN");
    return ok(parseChatReply(content));
  } catch {
    return fail(`Tidak dapat menghubungi ${label}. Coba kirim lagi.`, "NETWORK");
  }
}
