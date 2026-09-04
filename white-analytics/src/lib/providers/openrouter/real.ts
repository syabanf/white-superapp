import { ProviderError } from "@/lib/action-result";
import type { GenerateInsightInput, InsightModule, InsightProvider } from "./types";

const API_URL = "https://openrouter.ai/api/v1/chat/completions";
const DEFAULT_MODEL = "anthropic/claude-sonnet-4.5";
const TIMEOUT_MS = 90_000;

const BASE_RULES = `Kamu adalah kepala strategi (senior strategist) di sebuah digital agency Indonesia yang menulis analisa performa untuk klien.
Aturan wajib:
- Tulis dalam format markdown sederhana (heading "##", bullet "-", daftar bernomor, **bold**). Tanpa tabel, tanpa HTML.
- Gunakan angka-angka NYATA dari data JSON yang diberikan — jangan pernah mengarang angka.
- Nada: tajam, membumi, berorientasi aksi; seperti konsultan senior menulis untuk pemilik bisnis.
- Jangan pernah menyebut AI, model bahasa, otomasi, atau proses pembuatan analisa ini.
- Jangan menyapa pembaca dan jangan menambahkan penutup basa-basi.`;

const SECTION_CORE = `Struktur output WAJIB memakai heading berikut, dalam Bahasa Indonesia dan urutan ini:
## Ringkasan eksekutif
(1 paragraf padat: kondisi umum periode ini vs periode sebelumnya)
## Sorotan performa
(bullet — pencapaian terkuat dengan angka)
## Area perbaikan
(bullet — kelemahan atau risiko dengan angka)`;

const SECTION_RECS = `## Rekomendasi 30 hari
(TEPAT 5 butir bernomor 1.–5., masing-masing satu aksi konkret yang bisa dieksekusi dalam 30 hari)`;

const SYSTEM_PROMPTS: Record<InsightModule, string> = {
  OVERVIEW: `${BASE_RULES}

Fokus: ringkasan lintas kanal (social media, SEO organik, dan Meta Ads) — hubungkan ketiganya menjadi satu narasi funnel: jangkauan → trafik → konversi.
${SECTION_CORE}
${SECTION_RECS}`,
  SOCIAL: `${BASE_RULES}

Fokus: performa social media organik (Instagram/Facebook/TikTok) — pertumbuhan followers, engagement rate, jangkauan, dan pola konten yang bekerja (lihat topPosts).
${SECTION_CORE}
${SECTION_RECS}`,
  SEO: `${BASE_RULES}

Fokus: performa pencarian organik (Google Search Console) — klik, impresi, CTR, posisi rata-rata, kesehatan situs, dan kata kunci (topQueries).
${SECTION_CORE}
## Peluang kata kunci
(bullet — kata kunci posisi 4–15 dengan impresi tinggi yang layak didorong ke top 3, sebutkan posisi & impresinya)
${SECTION_RECS}`,
  ADS: `${BASE_RULES}

Fokus: performa Meta Ads — belanja, hasil, biaya per hasil, CTR/CPM/ROAS, dan perbandingan antar kampanye (campaigns).
${SECTION_CORE}
## Alokasi anggaran
(bullet — usulan realokasi anggaran antar kampanye berdasarkan efisiensi biaya per hasil, dengan angka)
${SECTION_RECS}`,
};

function mapStatus(status: number, detail: string): ProviderError {
  if (status === 401) return new ProviderError("openrouter", "TOKEN_EXPIRED", `OpenRouter menolak kredensial (401). ${detail}`);
  if (status === 403) return new ProviderError("openrouter", "PERMISSION", `Akses OpenRouter ditolak (403). ${detail}`);
  if (status === 429) return new ProviderError("openrouter", "RATE_LIMIT", `Batas permintaan OpenRouter tercapai (429). ${detail}`);
  if (status === 404) return new ProviderError("openrouter", "NOT_FOUND", `Model tidak ditemukan di OpenRouter (404). ${detail}`);
  return new ProviderError("openrouter", "UNKNOWN", `OpenRouter mengembalikan status ${status}. ${detail}`);
}

export function openRouterModel(): string {
  return process.env.OPENROUTER_MODEL || DEFAULT_MODEL;
}

export async function generateRealInsight(input: GenerateInsightInput): Promise<string> {
  const apiKey = process.env.OPENROUTER_API_KEY;
  if (!apiKey) throw new ProviderError("openrouter", "PERMISSION", "OPENROUTER_API_KEY belum dikonfigurasi.");

  const languageLine =
    input.language === "en"
      ? "Write the body text in English, but KEEP the section headings exactly as specified (in Indonesian)."
      : "Tulis seluruh isi dalam Bahasa Indonesia.";

  const userMessage = [
    `Klien: ${input.clientName}`,
    `Periode analisa: ${input.range.from} s.d. ${input.range.to} (${input.range.days} hari), dibandingkan ${input.range.prevFrom} s.d. ${input.range.prevTo}.`,
    `Catatan data: nilai *DeltaPct adalah perubahan % vs periode pembanding; engagementRate/ctr dalam persen; cpr/cpm dalam mata uang akun.`,
    languageLine,
    "",
    "Data KPI (JSON):",
    JSON.stringify(input.data),
  ].join("\n");

  let res: Response;
  try {
    res = await fetch(API_URL, {
      method: "POST",
      signal: AbortSignal.timeout(TIMEOUT_MS),
      headers: {
        Authorization: `Bearer ${apiKey}`,
        "Content-Type": "application/json",
        "HTTP-Referer": process.env.NEXT_PUBLIC_APP_URL ?? "http://localhost:3000",
        "X-Title": "WHITE Analytics",
      },
      body: JSON.stringify({
        model: openRouterModel(),
        max_tokens: 2500,
        temperature: 0.4,
        messages: [
          { role: "system", content: SYSTEM_PROMPTS[input.module] },
          { role: "user", content: userMessage },
        ],
      }),
    });
  } catch (cause) {
    throw new ProviderError("openrouter", "NETWORK", "Tidak dapat menghubungi OpenRouter (timeout/jaringan).", { cause });
  }

  if (!res.ok) {
    const detail = (await res.text().catch(() => "")).slice(0, 300);
    throw mapStatus(res.status, detail);
  }

  let json: { choices?: { message?: { content?: string } }[]; error?: { message?: string } };
  try {
    json = (await res.json()) as typeof json;
  } catch (cause) {
    throw new ProviderError("openrouter", "UNKNOWN", "Respons OpenRouter tidak dapat dibaca.", { cause });
  }
  const content = json.choices?.[0]?.message?.content?.trim();
  if (!content) {
    throw new ProviderError("openrouter", "UNKNOWN", json.error?.message ?? "OpenRouter tidak mengembalikan konten.");
  }
  return content;
}

export const realInsightProvider: InsightProvider = {
  id: "openrouter",
  get model() {
    return openRouterModel();
  },
  generateInsight: generateRealInsight,
};
