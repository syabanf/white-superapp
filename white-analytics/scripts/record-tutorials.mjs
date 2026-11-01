#!/usr/bin/env node
/**
 * Records the tutorial videos shown in /panduan and /panduan/erp.
 *
 * Each guide section gets one silent screen recording of the local app with
 * the seeded demo data: a title card, then scenes that open a page, point at
 * the control the guide talks about. Captions go to a WebVTT subtitle track
 * instead of the picture, so viewers can switch them off. Nothing is saved or
 * submitted; scenes only navigate, hover, switch tabs and scroll.
 *
 * Needs: `pnpm dev` on :3100, the Go gateway on :8380, the demo seed, and
 * playwright-core with its Chromium and ffmpeg (`npx playwright install chromium ffmpeg`).
 *
 *   node scripts/record-tutorials.mjs            # every section
 *   node scripts/record-tutorials.mjs ads erp-sdm  # only these
 *
 * Writes public/tutorials/<id>.webm + .vtt + .jpg and src/features/guide/videos.json.
 */
import { createRequire } from "node:module";
import { mkdir, readFile, rename, rm, writeFile } from "node:fs/promises";
import path from "node:path";

const require = createRequire(process.env.PLAYWRIGHT_CORE ?? import.meta.url);
const { chromium } = require("playwright-core");

const BASE = process.env.BASE_URL ?? "http://127.0.0.1:3100";
const ROOT = path.resolve(import.meta.dirname, "..");
const OUT = path.join(ROOT, "public/tutorials");
const MANIFEST = path.join(ROOT, "src/features/guide/videos.json");
const SIZE = { width: 1280, height: 720 };
const C = "/clients/kopi-nusantara";

/**
 * Scene: `go` opens a page, `spot` points at the first element matching a
 * Playwright selector, `click` presses a harmless control (a tab), `scroll`
 * moves the page by that many pixels, `say` is the subtitle.
 */
const SECTIONS = {
  mulai: [
    { go: "/", say: "Portofolio menampilkan semua klien dan performa terbarunya." },
    { spot: "text=Klien baru", say: "Mulai project dengan tombol Klien baru di sidebar." },
    { go: "/clients/new", say: "Pilih jenis project, isi nama klien, lalu hubungkan sumber data." },
    { go: C, say: "Dashboard klien langsung terbuka, dengan data demo bila sumber data belum terhubung." },
  ],
  periode: [
    { go: C, say: "Filter tanggal di kanan atas mengatur semua angka di halaman." },
    { spot: "header button:has-text('hari'), header button:has-text('Bulan')", say: "Pilih preset atau rentang kustom. Periode selalu berakhir kemarin." },
    { scroll: 400, say: "KPI, grafik, dan tabel selalu memakai periode yang sama." },
  ],
  social: [
    { go: `${C}/social`, say: "Social Media merangkum Instagram dan Facebook klien." },
    { scroll: 450, say: "Grafik pertumbuhan dan engagement mengikuti filter periode." },
    { scroll: 500, say: "Daftar post teratas membantu melihat konten yang paling berhasil." },
    { go: `${C}/social/competitors`, say: "Halaman Kompetitor membandingkan akun klien dengan pesaing." },
  ],
  seo: [
    { go: `${C}/seo`, say: "SEO menampilkan klik, impresi, CTR, dan posisi dari Search Console." },
    { go: `${C}/seo/keywords`, say: "Kata kunci: lihat peluang dari posisi dan impresi." },
    { go: `${C}/seo/pages`, say: "Halaman: performa tiap URL di hasil pencarian." },
    { go: `${C}/seo/audit`, say: "Audit situs memeriksa kecepatan dan kesehatan teknis." },
  ],
  ads: [
    { go: `${C}/ads`, say: "Meta Ads: belanja, hasil, dan biaya per hasil dalam satu layar." },
    { scroll: 450, say: "Peringatan otomatis muncul saat CPR naik, CTR turun, atau frekuensi di atas 3." },
    { go: `${C}/ads/campaigns`, say: "Campaign Builder menyusun kampanye dari objektif dan anggaran." },
  ],
  "konten-ai": [
    { go: `${C}/content/brand`, say: "Brand Kit menyimpan warna, font, tone, dan audiens untuk semua fitur AI." },
    { scroll: 500, say: "Isi do & don't, kata kunci, dan contoh caption supaya hasil AI on-brand." },
    { go: `${C}/content/studio`, say: "Studio Desain AI: tulis brief, pilih format, lalu buat desain." },
  ],
  laporan: [
    { go: `${C}/reports`, say: "Menu Laporan: pilih modul dan periode, lalu unduh PDF." },
    { click: "role=tab[name=/Riwayat/]", say: "Riwayat menyimpan laporan lama dan jadwal laporan otomatis." },
    { click: "role=tab[name=/Link klien/]", say: "Link klien membuat tautan read-only, dengan PIN bila perlu." },
  ],
  akses: [
    { go: "/admin/users", say: "Manajemen User mengatur siapa bisa masuk dan perannya." },
    { spot: "role=button[name=/Undang|Tambah|User baru/]", say: "Undang anggota tim, lalu atur peran dan izin modul ERP." },
    { go: `${C}/settings`, say: "Pengaturan Klien menentukan anggota tim yang memegang klien ini." },
  ],
  publikasi: [
    { go: `${C}/publish`, say: "Kalender Publikasi menampilkan semua post terjadwal." },
    { spot: "text=Buat post", say: "Buat post, pilih akun tujuan, tulis caption, lalu kirim untuk review." },
    { go: `${C}/publish/posts`, say: "Daftar post memperlihatkan status: draf, review, disetujui, terbit." },
    { go: `${C}/publish/media`, say: "Pustaka Media menyimpan gambar dan video untuk dipakai ulang." },
  ],
  konten: [
    { go: `${C}/content/plans`, say: "Editorial plan: rencana konten bulanan beserta brief-nya." },
    { go: `${C}/content/board`, say: "Papan konten: seret kartu dari Ide sampai Terbit." },
    { go: `${C}/content/design`, say: "Tugas desain: brief, revisi, dan persetujuan desain final." },
    { go: `${C}/content/assets`, say: "Aset menyimpan file brand dan hasil desain." },
    { go: `${C}/content/reports`, say: "Laporan konten merangkum produksi per periode." },
  ],
  "seo-suite": [
    { go: `${C}/seo/research`, say: "Riset Kata Kunci mencari ide dari satu kata kunci awal." },
    { go: `${C}/seo/rank`, say: "Peringkat melacak posisi kata kunci pilihan setiap hari." },
    { go: `${C}/seo/backlinks`, say: "Backlink memantau profil tautan yang mengarah ke situs." },
    { go: `${C}/seo/competitors`, say: "Kompetitor Domain membandingkan domain Anda dengan hingga 5 pesaing." },
  ],
  integrasi: [
    { go: "/setup", say: "Setup awal menghubungkan Meta, Google, dan layanan AI." },
    { scroll: 450, say: "Ikuti langkahnya satu per satu. Status tiap integrasi tampil di sini." },
  ],

  "erp-mulai": [
    { go: "/erp", say: "Beranda ERP: kartu Perlu perhatian dan ringkasan tiap modul." },
    { spot: "[data-sidebar=header] button", say: "Pindah antara Digital Marketing dan ERP lewat pemilih ruang kerja." },
    { scroll: 450, say: "Menu dan kartu tampil sesuai izin akun Anda." },
  ],
  "erp-keuangan": [
    { go: "/erp/finance", say: "Dasbor keuangan: kas, pendapatan, beban, dan tren." },
    { go: "/erp/finance/accounts", say: "Bagan akun: tambah akun atau impor dari XLSX." },
    { go: "/erp/finance/journals", say: "Jurnal: catat transaksi dengan debit dan kredit seimbang." },
    { go: "/erp/finance/cash-bank", say: "Kas & bank: kas masuk, kas keluar, dan transfer antar rekening." },
    { go: "/erp/finance/reports", say: "Laporan keuangan: neraca, laba rugi, arus kas, dan buku besar." },
  ],
  "erp-reimburse": [
    { go: "/erp/me/reimbursements", say: "Karyawan membuat klaim di Reimburse saya." },
    { spot: "role=button[name=/Klaim baru/]", say: "Klaim baru: kategori, tanggal, jumlah, dan bukti." },
    { go: "/erp/finance/reimbursements", say: "Finance menyetujui klaim, lalu menandainya dibayar." },
  ],
  "erp-sdm": [
    { go: "/erp/hr", say: "Dasbor SDM: jumlah karyawan, kehadiran, dan pengajuan menunggu." },
    { go: "/erp/hr/employees", say: "Karyawan: data lengkap, atasan langsung, dan dokumen." },
    { go: "/erp/hr/attendance", say: "Absensi: roster harian, rekap, shift, dan hari libur." },
    { go: "/erp/hr/leave", say: "Cuti disetujui dua tahap: atasan lalu HRD." },
    { go: "/erp/hr/contracts", say: "Kontrak kerja: aktifkan, perpanjang, atau konversi ke PKWTT." },
  ],
  "erp-payroll": [
    { go: "/erp/hr/payroll", say: "Payroll: lengkapi data gaji tiap karyawan." },
    { click: "role=tab[name=/Proses gaji/]", say: "Proses gaji: buat, hitung, setujui, lalu tandai dibayar." },
    { go: "/erp/me/payslips", say: "Karyawan melihat slip gaji di Layanan saya." },
  ],
  "erp-rekrutmen": [
    { go: "/erp/hr/recruitment", say: "Rekrutmen: ringkasan lowongan dan kandidat aktif." },
    { go: "/erp/hr/recruitment/jobs", say: "Lowongan: tayangkan posisi baru ke portal karier." },
    { go: "/erp/hr/recruitment/candidates", say: "Kandidat: saring lamaran dan jalankan penilaian." },
    { go: "/erp/hr/recruitment/pipeline", say: "Pipeline: seret kandidat antar tahap." },
    { go: "/erp/hr/recruitment/interviews", say: "Jadwal interview, lalu kirim penawaran." },
  ],
  "erp-kinerja": [
    { go: "/erp/performance", say: "Kinerja: ringkasan tugas, KPI, dan review tim." },
    { go: "/erp/performance/tasks", say: "Task tim: tugas berulang dengan penanggung jawab." },
    { go: "/erp/performance/kpi", say: "KPI: indikator, bobot departemen, target, dan scorecard." },
    { go: "/erp/performance/reviews", say: "Review kinerja per kuartal, ditambah feedback 360." },
  ],
  "erp-penjualan": [
    { go: "/erp/sales/leads", say: "Leads: tambah atau impor calon klien." },
    { go: "/erp/sales/pipeline", say: "Pipeline: seret deal antar tahap sampai menang." },
    { go: "/erp/sales/quotations", say: "Quotation: penawaran harga per deal dengan versi dan persetujuan." },
    { go: "/erp/sales/quotations/new", say: "Quotation baru: pilih pelanggan, lalu tarik dari deal." },
    { go: "/erp/sales/invoices", say: "Invoice: terbitkan dari termin dan pantau pembayaran." },
    { go: "/erp/sales/contracts", say: "Kontrak klien dikirim untuk tanda tangan, lalu onboarding." },
  ],
  "erp-penjualan-tools": [
    { go: "/erp/sales/pitch-decks", say: "Pitch deck: susun presentasi untuk calon klien." },
    { go: "/erp/sales/chatbot", say: "Chatbot menjawab calon klien dan mencatatnya sebagai lead." },
    { go: "/erp/sales/forms", say: "Formulir lead: bagikan tautan, isian masuk ke Leads." },
    { go: "/erp/sales/automation", say: "Otomasi penjualan menjalankan tindak lanjut sesuai aturan." },
  ],
  "erp-mandiri": [
    { go: "/erp/me", say: "Layanan saya: clock in, clock out, dan ringkasan pribadi." },
    { go: "/erp/me/leave", say: "Cuti saya: saldo, riwayat, dan pengajuan baru." },
    { go: "/erp/me/loans", say: "Pinjaman saya: kasbon atau pinjaman dengan cicilan dari gaji." },
    { go: "/erp/me/tasks", say: "Task saya dikelompokkan: terlambat, hari ini, mendatang." },
  ],
};

/** Title card, a visible cursor and a pulsing ring, injected into every page. */
const overlay = ([eyebrow, title]) => {
  const css = `
    #tut-cur{position:fixed;width:22px;height:22px;margin:-11px 0 0 -11px;border-radius:50%;z-index:2147483647;pointer-events:none;
      background:rgba(160,29,99,.35);border:2px solid #A01D63;opacity:0;transition:opacity .2s}
    #tut-ring{position:fixed;z-index:2147483646;pointer-events:none;border:3px solid #A01D63;border-radius:14px;opacity:0;
      box-shadow:0 0 0 6px rgba(160,29,99,.18);transition:all .35s}
    nextjs-portal{display:none!important}`;
  const mount = () => {
    if (document.getElementById("tut-cur")) return;
    const style = document.createElement("style");
    style.textContent = css;
    document.head.append(style);
    for (const id of ["tut-cur", "tut-ring"]) {
      const el = document.createElement("div");
      el.id = id;
      document.body.append(el);
    }
    addEventListener("mousemove", (e) => {
      const cur = document.getElementById("tut-cur");
      cur.style.left = `${e.clientX}px`;
      cur.style.top = `${e.clientY}px`;
      cur.style.opacity = "1";
    });
  };
  window.__tut = {
    mount,
    ring(r) {
      const ring = document.getElementById("tut-ring");
      Object.assign(ring.style, r
        ? { left: `${r.x - 6}px`, top: `${r.y - 6}px`, width: `${r.width + 12}px`, height: `${r.height + 12}px`, opacity: "1" }
        : { opacity: "0" });
    },
    uncard() {
      window.__tutUncarded = true;
      const card = document.getElementById("tut-card");
      if (!card) return;
      card.style.opacity = "0";
      setTimeout(() => card.remove(), 500);
    },
  };
  // The first page of a recording opens behind the title card. It goes on the
  // root element at once: streamed pages fire DOMContentLoaded only at the end.
  // about:blank has no sessionStorage: it shows the card while the first page loads.
  let carded = false;
  try {
    carded = !!sessionStorage.getItem("tut-carded");
    sessionStorage.setItem("tut-carded", "1");
  } catch {}
  if (!carded) {
    const card = document.createElement("div");
    card.id = "tut-card";
    card.style.cssText = "position:fixed;inset:0;z-index:2147483647;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:14px;background:linear-gradient(135deg,#A01D63,#2B2265);color:#fff;font-family:system-ui,sans-serif;transition:opacity .5s";
    card.innerHTML = `<small style="font:600 15px system-ui;letter-spacing:.2em;text-transform:uppercase;opacity:.75"></small><b style="font:800 42px/1.2 system-ui;max-width:960px;text-align:center"></b>`;
    card.querySelector("small").textContent = eyebrow;
    card.querySelector("b").textContent = title;
    // <html> may not exist yet, and React hydration drops foreign nodes from it:
    // put the card back on every change until uncard().
    const keep = () => card.isConnected || window.__tutUncarded || document.documentElement?.append(card);
    keep();
    new MutationObserver(keep).observe(document, { childList: true, subtree: true });
  }
  if (document.readyState === "loading") addEventListener("DOMContentLoaded", mount);
  else mount();
};

const pause = (page, ms) => page.waitForTimeout(ms);
// A client-side redirect can swap the document mid-call; the next scene mounts again.
const tut = (page, fn, ...args) => page.evaluate(([fn, args]) => window.__tut?.[fn](...args), [fn, args]).catch(() => {});

async function open(page, url) {
  // Some pages poll and never go network-idle, so the heading is what counts.
  await page.goto(BASE + url, { waitUntil: "domcontentloaded", timeout: 45_000 }).catch(() => {});
  await page.waitForSelector("main h1", { timeout: 20_000 }).catch(() => {});
  await page.waitForLoadState("networkidle", { timeout: 5_000 }).catch(() => {});
  await tut(page, "mount");
  // Let the page finish its entrance animation before the caption lands.
  await page
    .waitForFunction(() => document.querySelector("main h1") && document.getAnimations().every((a) => a.playState !== "running" || a.effect?.getTiming().iterations === Infinity), null, { timeout: 8_000 })
    .catch(() => {});
  await pause(page, 500);
}

/** Moves the cursor to an element and rings it; returns false when the element is missing. */
async function point(page, selector) {
  const el = page.locator(selector).first();
  const box = await el.boundingBox({ timeout: 3_000 }).catch(() => null);
  if (!box) return false;
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2, { steps: 25 });
  await tut(page, "ring", box);
  return true;
}

const stamp = (ms) => new Date(ms).toISOString().slice(11, 23);
const vtt = (cues) => `WEBVTT\n\n${cues.map((c, i) => `${i + 1}\n${stamp(c.start)} --> ${stamp(c.end)}\n${c.text}\n`).join("\n")}`;

async function record(browser, state, id, title, eyebrow) {
  const dir = path.join(OUT, `.tmp-${id}`);
  const context = await browser.newContext({
    viewport: SIZE,
    storageState: state,
    reducedMotion: "no-preference",
    recordVideo: { dir, size: SIZE },
  });
  await context.addInitScript(overlay, [eyebrow, title]);
  const page = await context.newPage();
  const started = Date.now();
  const cues = [];
  const subtitle = (text) => {
    const at = Date.now() - started;
    if (cues.length) cues.at(-1).end = at;
    if (text) cues.push({ start: at, end: at, text });
  };
  let poster = false;

  const scenes = SECTIONS[id];
  await page.setContent("<body style='margin:0'></body>");
  await open(page, scenes[0].go);
  await pause(page, 1600);
  await tut(page, "uncard");
  await pause(page, 600);

  for (const [i, s] of scenes.entries()) {
    if (s.go && i > 0) await open(page, s.go);
    await tut(page, "ring", null);
    if (s.scroll) await page.mouse.wheel(0, s.scroll);
    if (s.click && (await point(page, s.click))) {
      await page.locator(s.click).first().click({ timeout: 3_000 }).catch(() => {});
      await pause(page, 800);
      await tut(page, "ring", null);
    }
    if (s.spot) await point(page, s.spot);
    subtitle(s.say);
    if (!poster) {
      await page.screenshot({ path: path.join(OUT, `${id}.jpg`), type: "jpeg", quality: 70 });
      poster = true;
    }
    await pause(page, Math.max(3200, s.say.length * 60));
  }
  await pause(page, 500);
  subtitle(null);

  const seconds = Math.round((Date.now() - started) / 1000);
  const video = page.video();
  await context.close();
  await rename(await video.path(), path.join(OUT, `${id}.webm`));
  await writeFile(path.join(OUT, `${id}.vtt`), vtt(cues));
  await rm(dir, { recursive: true, force: true });
  return seconds;
}

const only = process.argv.slice(2);
const ids = only.length ? only : Object.keys(SECTIONS);
const unknown = ids.filter((id) => !SECTIONS[id]);
if (unknown.length) throw new Error(`No scenes for: ${unknown.join(", ")}`);

await mkdir(OUT, { recursive: true });
const browser = await chromium.launch();
const login = await browser.newPage({ viewport: SIZE });
await login.goto(`${BASE}/login`);
await login.fill("#email", process.env.TUTORIAL_EMAIL ?? "admin@mixist.id");
await login.fill("#password", process.env.TUTORIAL_PASSWORD ?? "admin12345");
await Promise.all([login.waitForURL((u) => !u.pathname.startsWith("/login")), login.click("button[type=submit]")]);
// Title cards read the section titles off the tutorial pages, so they always match the guide.
const titles = {};
for (const url of ["/panduan", "/panduan/erp"]) {
  await login.goto(BASE + url, { waitUntil: "networkidle" });
  Object.assign(titles, await login.$$eval("[id] > header h2", (hs) => Object.fromEntries(hs.map((h) => [h.closest("[id]").id, h.textContent.trim()]))));
}
// Dev builds compile a route on first visit; open each page once so recordings never show that wait.
for (const url of new Set(ids.flatMap((id) => SECTIONS[id].map((s) => s.go).filter(Boolean)))) {
  await login.goto(BASE + url, { waitUntil: "networkidle", timeout: 90_000 }).catch(() => {});
}
const state = await login.context().storageState();
await login.close();

// The guide imports this file, so it is rewritten after every video and never removed.
const manifest = JSON.parse(await readFile(MANIFEST, "utf8").catch(() => "{}"));
for (const id of ids) {
  const eyebrow = id.startsWith("erp-") ? "Tutorial ERP" : "Tutorial Digital Marketing";
  manifest[id] = await record(browser, state, id, titles[id] ?? id, eyebrow);
  const sorted = Object.fromEntries(Object.entries(manifest).sort(([a], [b]) => a.localeCompare(b)));
  await writeFile(MANIFEST, `${JSON.stringify(sorted, null, 2)}\n`);
  console.log(`${id}: ${manifest[id]}s`);
}
await browser.close();
