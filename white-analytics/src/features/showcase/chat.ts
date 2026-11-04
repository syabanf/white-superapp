import type { ChatScenario, ChatStep } from "./data";

export type ChatReply = { reply: string; effect?: string };

const RULES = `Kamu Asisten Mixist yang bekerja lewat WhatsApp, dalam simulasi untuk demo produk.
Aturan:
- Tanggapi isi pesan terakhir apa adanya. Jangan berpura-pura pengguna menulis hal lain.
- Bahasa Indonesia yang singkat seperti chat WhatsApp, tanpa markdown.
- Pakai hanya fakta dari tugas di bawah dan isi obrolan. Jangan mengarang angka, nama, atau tanggal.
- Kalau pesan di luar tugas, jawab sopan lalu arahkan kembali ke tugas.
- Balas hanya JSON: {"reply": "isi balasan", "effect": "satu kalimat tentang yang tercatat di dashboard"}. Isi effect dengan null kalau tidak ada yang tercatat.`;

/** Turns the room into chat messages. The opening comes from the scenario, never from the browser. */
export function buildChatMessages(scenario: ChatScenario, turns: ChatStep[]) {
  return [
    { role: "system", content: `${RULES}\n\nRuang: ${scenario.room}\nTugas: ${scenario.context}` },
    ...[...scenario.opening, ...turns].map((m) =>
      m.from === "bot" ? { role: "assistant", content: m.text } : { role: "user", content: `${m.name}: ${m.text}` },
    ),
  ];
}

/** Reads the model's JSON. A model that ignores the format still gets its text shown. */
export function parseChatReply(content: string): ChatReply {
  const text = content.trim();
  const json = text.slice(text.indexOf("{"), text.lastIndexOf("}") + 1);
  try {
    const { reply, effect } = JSON.parse(json) as { reply?: unknown; effect?: unknown };
    if (typeof reply === "string" && reply.trim()) {
      return { reply: reply.trim(), effect: typeof effect === "string" && effect.trim() ? effect.trim() : undefined };
    }
  } catch {
    // Falls through to the plain text below.
  }
  return { reply: text };
}
