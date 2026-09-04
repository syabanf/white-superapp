/**
 * Static seed content — Indonesian brands, keywords, captions, ad names.
 * Kept separate from seed.ts so the generator stays readable.
 */

export type SeedClient = {
  name: string;
  slug: string;
  description: string;
  industry: string;
  websiteUrl: string;
  siteUrl: string; // GSC property
  ga4PropertyId: string;
  own: { ig: string; igName: string; fb: string; fbName: string; tiktok: string; tiktokName: string };
  competitors: { username: string; name: string; bio: string; followers: number }[];
  baseFollowers: { ig: number; fb: number; tiktok: number };
  dailyGain: { ig: number; fb: number; tiktok: number };
  queries: string[];
  brandQueries: string[];
  pages: string[];
  countries: { key: string; share: number }[];
  seoBase: { clicks: number; impressions: number; position: number; trend: number };
  ga4Base: { sessions: number; organicShare: number; conv: number };
  ads: {
    accountId: string;
    accountName: string;
    campaigns: { name: string; objective: string; resultType: string; dailyBudget: number; status: string; startDay: number; endDay?: number; cpr: number; ctr: number; adSets: { name: string; ads: string[] }[] }[];
  };
  captions: string[];
  hashtags: string[];
  auditUrl: string;
  auditScores: { performance: number; seo: number; accessibility: number; bestPractices: number }[];
};

export const SEED_CLIENTS: SeedClient[] = [
  {
    name: "Kopi Nusantara",
    slug: "kopi-nusantara",
    description: "Jaringan kedai kopi lokal dengan 12 cabang di Jabodetabek. Fokus: brand awareness, traffic ke cabang, dan penjualan online.",
    industry: "F&B",
    websiteUrl: "https://kopinusantara.id",
    siteUrl: "https://kopinusantara.id/",
    ga4PropertyId: "properties/398211004",
    own: {
      ig: "kopinusantara",
      igName: "Kopi Nusantara ☕",
      fb: "KopiNusantaraID",
      fbName: "Kopi Nusantara",
      tiktok: "kopinusantara.id",
      tiktokName: "Kopi Nusantara",
    },
    competitors: [
      { username: "kopi.senja", name: "Kopi Senja", bio: "Kopi lokal, rasa dunia. 40+ cabang.", followers: 186_000 },
      { username: "ngopi.dulu", name: "Ngopi Dulu Coffee", bio: "Kopi susu gula aren #1 pilihan anak Jaksel", followers: 92_400 },
      { username: "arabika.house", name: "Arabika House", bio: "Specialty coffee roastery · Bandung", followers: 41_800 },
    ],
    baseFollowers: { ig: 54_200, fb: 21_300, tiktok: 38_900 },
    dailyGain: { ig: 38, fb: 6, tiktok: 55 },
    queries: [
      "kopi susu gula aren", "kedai kopi jakarta", "biji kopi arabika", "cold brew coffee", "kopi kekinian", "franchise kopi",
      "kopi tubruk", "harga kopi susu", "kopi terdekat", "kopi enak jakarta selatan", "coffee shop instagramable", "kopi arabika gayo",
      "kopi robusta lampung", "cara membuat kopi susu gula aren", "resep es kopi susu", "kopi tanpa gula", "manfaat kopi hitam",
      "kopi untuk diet", "kopi bubuk premium", "kopi kemasan botol", "kopi literan", "promo kopi hari ini", "kopi 1 liter harga",
      "kopi drip bag", "biji kopi grosir", "supplier biji kopi", "kopi single origin", "kopi flores bajawa", "kopi toraja",
      "kopi jawa preanger", "kopi papua wamena", "kopi bali kintamani", "kopi aceh gayo harga", "kopi sachet enak",
      "coffee shop jakarta selatan", "tempat ngopi nyaman", "kopi susu terdekat buka 24 jam", "kopi es kekinian",
      "beda arabika dan robusta", "cara seduh v60",
    ],
    brandQueries: ["kopi nusantara", "kopi nusantara menu", "kopi nusantara harga", "kopi nusantara cabang", "kopi nusantara terdekat", "kopi nusantara promo"],
    pages: ["/", "/menu", "/lokasi", "/promo", "/blog/kopi-susu-gula-aren-resep", "/blog/beda-arabika-robusta", "/blog/cara-seduh-v60", "/produk/biji-kopi-gayo", "/produk/kopi-literan", "/franchise", "/tentang", "/blog/manfaat-kopi-hitam", "/blog/kopi-single-origin-indonesia", "/karier", "/kontak"],
    countries: [
      { key: "idn", share: 0.9 },
      { key: "sgp", share: 0.04 },
      { key: "mys", share: 0.03 },
      { key: "aus", share: 0.02 },
      { key: "usa", share: 0.01 },
    ],
    seoBase: { clicks: 380, impressions: 14_500, position: 11.8, trend: 0.0028 },
    ga4Base: { sessions: 1_450, organicShare: 0.46, conv: 0.021 },
    ads: {
      accountId: "act_958133483878991",
      accountName: "Kopi Nusantara Ads",
      campaigns: [
        {
          name: "[Prospecting] Kopi Susu Aren – Traffic",
          objective: "OUTCOME_TRAFFIC",
          resultType: "link_click",
          dailyBudget: 350_000,
          status: "ACTIVE",
          startDay: 0,
          cpr: 1_650,
          ctr: 1.9,
          adSets: [
            { name: "Jaksel 25-34 · Interest Kopi", ads: ["Video Kopi Susu Aren 15s", "Carousel Menu Signature"] },
            { name: "Jabodetabek 18-24 · Lookalike 1%", ads: ["Reels UGC Review", "Static Promo Beli 2 Gratis 1"] },
          ],
        },
        {
          name: "[Retargeting] Promo Payday – Sales",
          objective: "OUTCOME_SALES",
          resultType: "purchase",
          dailyBudget: 250_000,
          status: "ACTIVE",
          startDay: 10,
          cpr: 38_000,
          ctr: 2.6,
          adSets: [
            { name: "Website Visitors 30D", ads: ["Promo Payday 25%", "Bundle Kopi Literan"] },
            { name: "IG Engagers 90D", ads: ["Testimoni Pelanggan", "Countdown Promo"] },
          ],
        },
        {
          name: "[Awareness] Brand Video – Reach",
          objective: "OUTCOME_AWARENESS",
          resultType: "reach",
          dailyBudget: 200_000,
          status: "PAUSED",
          startDay: 0,
          endDay: 70,
          cpr: 12,
          ctr: 0.6,
          adSets: [{ name: "Broad Jabodetabek 18-45", ads: ["Brand Film 30s", "Cerita Petani Kopi"] }],
        },
        {
          name: "[Leads] Franchise – Lead Form",
          objective: "OUTCOME_LEADS",
          resultType: "lead",
          dailyBudget: 300_000,
          status: "ACTIVE",
          startDay: 40,
          cpr: 62_000,
          ctr: 1.1,
          adSets: [{ name: "Pengusaha 30-50 · Interest Bisnis", ads: ["Peluang Franchise 2026", "ROI Franchise Kopi"] }],
        },
      ],
    },
    captions: [
      "Pagi yang sempurna dimulai dari secangkir Kopi Susu Aren favoritmu ☕✨",
      "Biji Arabika Gayo pilihan, disangrai fresh setiap minggu. Rasakan bedanya!",
      "Promo payday! Beli 2 gratis 1 untuk semua menu es kopi. Berlaku sampai hari Minggu.",
      "Behind the scenes: proses roasting di roastery kami di Bogor 🔥",
      "Cabang baru di Bintaro sudah buka! Mampir yuk, ada diskon 30% minggu ini.",
      "Tips: simpan biji kopi di wadah kedap udara, jauhkan dari cahaya matahari.",
      "Kolaborasi spesial bersama @roti.enak — Es Kopi Aren x Roti Bakar Cokelat 🍞",
      "Cerita petani: Pak Wayan dari Kintamani dan kebun kopinya yang berusia 40 tahun.",
      "Cold brew 24 jam, manis alami tanpa gula tambahan. Sudah coba?",
      "Kamu tim kopi hitam atau kopi susu? Komen di bawah 👇",
      "Menu baru: Kopi Pandan Latte 🌿 Hanya di cabang Jaksel & Depok.",
      "Terima kasih 50.000 followers! Giveaway voucher Rp 500 ribu — cek caption 🎁",
      "Kopi literan 1L cocok untuk WFH seharian. Pesan via website, gratis ongkir Jabodetabek.",
      "Weekend vibes di rooftop cabang Kemang 🌆",
      "Cara seduh V60 ala barista kami — swipe untuk langkahnya ➡️",
    ],
    hashtags: ["#kopinusantara", "#kopisusuaren", "#kopikekinian", "#ngopi", "#coffeeshopjakarta", "#kopilokal", "#specialtycoffee", "#kopiindonesia"],
    auditUrl: "https://kopinusantara.id/",
    auditScores: [
      { performance: 58, seo: 82, accessibility: 79, bestPractices: 83 },
      { performance: 63, seo: 88, accessibility: 81, bestPractices: 87 },
      { performance: 71, seo: 91, accessibility: 86, bestPractices: 92 },
      { performance: 76, seo: 94, accessibility: 88, bestPractices: 92 },
    ],
  },
  {
    name: "Adiharjo Property",
    slug: "adiharjo-property",
    description: "Developer perumahan cluster premium di Tangerang Selatan & Bogor. Fokus: lead generation KPR dan kunjungan show unit.",
    industry: "Properti",
    websiteUrl: "https://adiharjo.co.id",
    siteUrl: "sc-domain:adiharjo.co.id",
    ga4PropertyId: "properties/401877210",
    own: {
      ig: "adiharjo.property",
      igName: "Adiharjo Property",
      fb: "AdiharjoProperty",
      fbName: "Adiharjo Property",
      tiktok: "adiharjo.property",
      tiktokName: "Adiharjo Property",
    },
    competitors: [
      { username: "griya.asri.residence", name: "Griya Asri Residence", bio: "Hunian asri di BSD. Cluster baru launching!", followers: 27_300 },
      { username: "summarecon.serpong.info", name: "Info Serpong Property", bio: "Update properti Serpong & BSD", followers: 63_100 },
    ],
    baseFollowers: { ig: 18_700, fb: 34_600, tiktok: 9_800 },
    dailyGain: { ig: 14, fb: 9, tiktok: 22 },
    queries: [
      "rumah dijual tangerang selatan", "perumahan bsd", "kpr rumah", "rumah minimalis", "cluster baru tangerang", "rumah subsidi",
      "harga rumah 2026", "developer perumahan", "rumah 2 lantai", "cicilan rumah", "simulasi kpr", "rumah dekat stasiun",
      "townhouse jakarta", "rumah dijual depok", "investasi properti", "tips beli rumah pertama", "kpr syariah", "rumah siap huni",
      "cluster premium", "rumah type 36", "rumah type 45", "biaya notaris jual beli rumah", "pajak jual beli rumah",
      "brosur perumahan", "promo dp 0%", "rumah tanpa dp", "cara mengajukan kpr", "bunga kpr terendah", "rumah dijual bintaro",
      "perumahan serpong", "ruko dijual", "kavling tanah dijual", "rumah dijual bogor", "developer terpercaya", "smart home perumahan",
      "rumah dijual bsd city", "perumahan dekat tol", "rumah 3 kamar tidur", "cluster bogor", "rumah dijual ciputat",
    ],
    brandQueries: ["adiharjo property", "adiharjo residence", "adiharjo cluster", "adiharjo bogor", "adiharjo serpong harga", "adiharjo property review"],
    pages: ["/", "/proyek/adiharjo-residence-serpong", "/proyek/adiharjo-hills-bogor", "/simulasi-kpr", "/blog/tips-beli-rumah-pertama", "/blog/cara-mengajukan-kpr", "/blog/biaya-notaris-jual-beli", "/promo", "/tentang", "/kontak", "/blog/kpr-syariah-vs-konvensional", "/proyek", "/blog/perumahan-dekat-tol-serpong", "/karier", "/galeri"],
    countries: [
      { key: "idn", share: 0.95 },
      { key: "sgp", share: 0.02 },
      { key: "mys", share: 0.015 },
      { key: "aus", share: 0.01 },
      { key: "jpn", share: 0.005 },
    ],
    seoBase: { clicks: 210, impressions: 9_800, position: 14.2, trend: 0.0035 },
    ga4Base: { sessions: 820, organicShare: 0.52, conv: 0.034 },
    ads: {
      accountId: "act_120933412009",
      accountName: "Adiharjo Property Ads",
      campaigns: [
        {
          name: "[Leads] Adiharjo Residence Serpong – Lead Form",
          objective: "OUTCOME_LEADS",
          resultType: "lead",
          dailyBudget: 600_000,
          status: "ACTIVE",
          startDay: 0,
          cpr: 48_500,
          ctr: 1.3,
          adSets: [
            { name: "Tangsel 28-45 · Interest KPR", ads: ["Video Show Unit 30s", "Carousel Tipe Rumah"] },
            { name: "Jakarta 30-50 · Lookalike Leads", ads: ["Promo DP 0% Static", "Testimoni Penghuni"] },
          ],
        },
        {
          name: "[Leads] Adiharjo Hills Bogor – WhatsApp",
          objective: "OUTCOME_ENGAGEMENT",
          resultType: "messaging_conversation_started",
          dailyBudget: 400_000,
          status: "ACTIVE",
          startDay: 25,
          cpr: 31_000,
          ctr: 1.6,
          adSets: [{ name: "Bogor & Depok 25-40", ads: ["Reels View Bukit", "Static Cicilan 3 Jutaan"] }],
        },
        {
          name: "[Traffic] Simulasi KPR – Website",
          objective: "OUTCOME_TRAFFIC",
          resultType: "link_click",
          dailyBudget: 250_000,
          status: "ACTIVE",
          startDay: 0,
          cpr: 2_100,
          ctr: 1.8,
          adSets: [{ name: "Broad Jabodetabek 25-45", ads: ["Kalkulator KPR Gratis", "Berapa Cicilan Rumahmu?"] }],
        },
        {
          name: "[Awareness] Grand Launching Hills – Reach",
          objective: "OUTCOME_AWARENESS",
          resultType: "reach",
          dailyBudget: 300_000,
          status: "PAUSED",
          startDay: 20,
          endDay: 45,
          cpr: 9,
          ctr: 0.5,
          adSets: [{ name: "Bogor Raya 25-55", ads: ["Teaser Grand Launching", "Drone View Kawasan"] }],
        },
      ],
    },
    captions: [
      "Grand Launching Adiharjo Hills Bogor! Cluster asri di kaki bukit, 15 menit dari tol. Cicilan mulai 3 jutaan/bulan 🏡",
      "Show unit tipe 45 sudah bisa dikunjungi setiap hari, 09.00–17.00. Reservasi via link di bio.",
      "Tips: hitung rasio cicilan maksimal 30% dari penghasilan sebelum ajukan KPR.",
      "Progress pembangunan Adiharjo Residence Serpong tahap 2 — sudah 80%! 🚧",
      "Promo DP 0% & free biaya KPR sampai akhir bulan. Slot terbatas.",
      "Selamat kepada Bapak Rudi & keluarga yang serah terima kunci hari ini 🎉🔑",
      "Kenapa pilih cluster dekat tol? Hemat waktu, nilai investasi naik. Swipe ➡️",
      "Virtual tour 360° tipe 60 — link di bio 📱",
      "Fasilitas: kolam renang, jogging track, taman bermain, dan one-gate system 24 jam.",
      "Q&A: KPR syariah vs konvensional, mana yang cocok untukmu?",
      "Open house akhir pekan ini! Ada doorprize dan konsultasi KPR gratis.",
      "Adiharjo Property meraih penghargaan Developer Terpercaya 2026 🏆",
    ],
    hashtags: ["#adiharjoproperty", "#rumahdijual", "#perumahanserpong", "#kpr", "#rumahminimalis", "#clusterbaru", "#propertitangsel", "#investasiproperti"],
    auditUrl: "https://adiharjo.co.id/",
    auditScores: [
      { performance: 44, seo: 74, accessibility: 70, bestPractices: 75 },
      { performance: 49, seo: 79, accessibility: 72, bestPractices: 79 },
      { performance: 55, seo: 85, accessibility: 78, bestPractices: 83 },
      { performance: 61, seo: 88, accessibility: 80, bestPractices: 87 },
    ],
  },
  {
    name: "Bali Villa Escapes",
    slug: "bali-villa-escapes",
    description: "Manajemen & pemasaran 28 villa privat di Seminyak, Canggu, Ubud, dan Uluwatu. Fokus: booking langsung (direct booking) dari pasar Australia, Singapura & domestik.",
    industry: "Hospitality",
    websiteUrl: "https://balivillaescapes.com",
    siteUrl: "https://balivillaescapes.com/",
    ga4PropertyId: "properties/377120556",
    own: {
      ig: "balivillaescapes",
      igName: "Bali Villa Escapes",
      fb: "BaliVillaEscapes",
      fbName: "Bali Villa Escapes",
      tiktok: "balivillaescapes",
      tiktokName: "Bali Villa Escapes",
    },
    competitors: [
      { username: "villafinder.bali", name: "Villa Finder Bali", bio: "Hand-picked luxury villas in Bali", followers: 141_000 },
      { username: "canggu.villas", name: "Canggu Villas", bio: "Your home in Canggu 🌴", followers: 58_700 },
      { username: "ubud.hideaways", name: "Ubud Hideaways", bio: "Jungle villas & retreats in Ubud", followers: 33_200 },
    ],
    baseFollowers: { ig: 76_400, fb: 45_100, tiktok: 24_600 },
    dailyGain: { ig: 52, fb: 11, tiktok: 40 },
    queries: [
      "villa in bali", "bali villa rental", "private pool villa ubud", "seminyak villa", "canggu villa for rent", "luxury villa bali",
      "villa uluwatu cliff", "honeymoon villa bali", "family villa bali", "villa with chef bali", "monthly villa rental bali",
      "sewa villa bali", "villa murah bali", "villa seminyak private pool", "villa canggu murah", "villa ubud", "villa nusa dua",
      "villa jimbaran", "villa sanur", "bali wedding villa", "villa for digital nomads bali", "long term rental canggu",
      "bali villa with ocean view", "cheap villa bali", "villa 4 bedroom seminyak", "villa 2 bedroom canggu", "bali villa deals",
      "best area to stay in bali", "bali villa booking", "villa bali harga", "sewa villa canggu harian", "villa bali untuk keluarga",
      "villa bali dekat pantai", "villa bali kolam pribadi", "villa uluwatu", "villa canggu 3 bedroom", "villa bali murah 2026",
      "airbnb alternative bali", "bali villa staff", "villa seminyak 3 bedroom",
    ],
    brandQueries: ["bali villa escapes", "bali villa escapes review", "bali villa escapes canggu", "bali villa escapes seminyak", "bali villa escapes ubud", "balivillaescapes"],
    pages: ["/", "/villas", "/villas/seminyak", "/villas/canggu", "/villas/ubud", "/villas/uluwatu", "/blog/best-area-to-stay-in-bali", "/blog/honeymoon-villas-bali", "/blog/family-villas-with-chef", "/deals", "/about", "/contact", "/blog/long-term-rental-canggu", "/blog/villa-vs-hotel-bali", "/weddings"],
    countries: [
      { key: "aus", share: 0.34 },
      { key: "idn", share: 0.22 },
      { key: "sgp", share: 0.12 },
      { key: "usa", share: 0.1 },
      { key: "gbr", share: 0.08 },
      { key: "deu", share: 0.05 },
      { key: "ind", share: 0.05 },
      { key: "mys", share: 0.04 },
    ],
    seoBase: { clicks: 640, impressions: 31_000, position: 9.6, trend: 0.0022 },
    ga4Base: { sessions: 2_900, organicShare: 0.58, conv: 0.012 },
    ads: {
      accountId: "act_2278812331",
      accountName: "Bali Villa Escapes Ads",
      campaigns: [
        {
          name: "[Sales] Direct Booking – AU/SG",
          objective: "OUTCOME_SALES",
          resultType: "purchase",
          dailyBudget: 900_000,
          status: "ACTIVE",
          startDay: 0,
          cpr: 410_000,
          ctr: 1.4,
          adSets: [
            { name: "Australia 28-55 · Travel Intent", ads: ["Seminyak Sunset Video", "Canggu Family Villa Carousel"] },
            { name: "Singapore 25-45 · Weekend Getaway", ads: ["Ubud Jungle Villa Reels", "Deals Static 20% Off"] },
          ],
        },
        {
          name: "[Retargeting] Abandoned Booking – Sales",
          objective: "OUTCOME_SALES",
          resultType: "purchase",
          dailyBudget: 350_000,
          status: "ACTIVE",
          startDay: 5,
          cpr: 260_000,
          ctr: 2.9,
          adSets: [{ name: "Booking Page Visitors 14D", ads: ["Complete Your Booking", "Free Airport Transfer"] }],
        },
        {
          name: "[Traffic] Blog Content – Domestic",
          objective: "OUTCOME_TRAFFIC",
          resultType: "link_click",
          dailyBudget: 200_000,
          status: "ACTIVE",
          startDay: 0,
          cpr: 1_350,
          ctr: 2.2,
          adSets: [{ name: "Jakarta & Surabaya 25-40", ads: ["Villa vs Hotel Bali", "Best Area to Stay in Bali"] }],
        },
        {
          name: "[Awareness] Wedding Villas – Reach",
          objective: "OUTCOME_AWARENESS",
          resultType: "reach",
          dailyBudget: 250_000,
          status: "PAUSED",
          startDay: 30,
          endDay: 90,
          cpr: 15,
          ctr: 0.7,
          adSets: [{ name: "AU/SG Engaged Couples", ads: ["Wedding Villa Showreel", "Cliff Wedding Uluwatu"] }],
        },
      ],
    },
    captions: [
      "Golden hour at Villa Sawah, Canggu 🌅 Private pool, rice-field views, 5 minutes to Berawa Beach.",
      "Wake up in the jungle: Villa Alas, Ubud — now open for bookings. Link in bio.",
      "Long-stay deal: 30% off monthly rentals in Canggu until end of season 🌴",
      "Our villa chefs prepare fresh Balinese breakfast every morning. Nasi goreng, anyone? 🍳",
      "Honeymoon in Uluwatu: cliff-top infinity pool, sunset dinner, and total privacy 💛",
      "Meet Made, villa manager at Seminyak — 12 years of making guests feel at home.",
      "Family-friendly villas with kids' pools & baby cots. Tag someone who needs a Bali trip 👇",
      "Villa vs hotel — why 8 out of 10 of our guests never go back to hotels. Swipe ➡️",
      "New listing: 4-bedroom villa in Seminyak, 200 m from Eat Street 🍽️",
      "Wedding season is here 💍 Book your dream ceremony at Villa Karang, Uluwatu.",
      "Rainy-season magic in Ubud 🌧️ Cozy, green, and 25% off.",
      "Guest review: “Best villa experience in Bali — the staff went above and beyond.” ⭐⭐⭐⭐⭐",
    ],
    hashtags: ["#balivillaescapes", "#balivilla", "#canggu", "#seminyak", "#ubud", "#uluwatu", "#balitravel", "#privatepoolvilla"],
    auditUrl: "https://balivillaescapes.com/",
    auditScores: [
      { performance: 52, seo: 86, accessibility: 84, bestPractices: 88 },
      { performance: 57, seo: 90, accessibility: 86, bestPractices: 90 },
      { performance: 66, seo: 93, accessibility: 89, bestPractices: 92 },
      { performance: 72, seo: 96, accessibility: 91, bestPractices: 96 },
    ],
  },
];

export const ISSUE_CATALOG: { code: string; severity: "ERROR" | "WARNING" | "NOTICE"; message: string }[] = [
  { code: "MISSING_TITLE", severity: "ERROR", message: "Halaman tidak memiliki tag <title>." },
  { code: "TITLE_TOO_LONG", severity: "WARNING", message: "Title lebih dari 60 karakter dan berpotensi terpotong di hasil pencarian." },
  { code: "TITLE_TOO_SHORT", severity: "WARNING", message: "Title kurang dari 30 karakter." },
  { code: "MISSING_META_DESCRIPTION", severity: "WARNING", message: "Meta description tidak ditemukan." },
  { code: "META_DESCRIPTION_TOO_LONG", severity: "NOTICE", message: "Meta description lebih dari 160 karakter." },
  { code: "MISSING_H1", severity: "ERROR", message: "Halaman tidak memiliki heading H1." },
  { code: "MULTIPLE_H1", severity: "WARNING", message: "Halaman memiliki lebih dari satu H1." },
  { code: "IMG_MISSING_ALT", severity: "WARNING", message: "Gambar tanpa atribut alt." },
  { code: "BROKEN_INTERNAL_LINK", severity: "ERROR", message: "Tautan internal mengarah ke halaman 404." },
  { code: "MISSING_CANONICAL", severity: "NOTICE", message: "Tag canonical tidak ditemukan." },
  { code: "NOINDEX", severity: "WARNING", message: "Halaman diberi robots noindex." },
  { code: "THIN_CONTENT", severity: "WARNING", message: "Konten kurang dari 300 kata." },
  { code: "SLOW_RESPONSE", severity: "WARNING", message: "Waktu respons server lebih dari 1,5 detik." },
  { code: "MISSING_HREFLANG", severity: "NOTICE", message: "Halaman multibahasa tanpa hreflang." },
  { code: "NO_STRUCTURED_DATA", severity: "NOTICE", message: "Tidak ada structured data (JSON-LD)." },
  { code: "REDIRECT_CHAIN", severity: "WARNING", message: "Rantai redirect lebih dari satu lompatan." },
  { code: "MIXED_CONTENT", severity: "ERROR", message: "Aset dimuat melalui HTTP pada halaman HTTPS." },
];

export const AGES = ["18-24", "25-34", "35-44", "45-54", "55-64", "65+"];
export const AGE_WEIGHTS = [0.14, 0.38, 0.27, 0.13, 0.06, 0.02];
export const GENDERS = ["female", "male", "unknown"];
export const GENDER_WEIGHTS = [0.55, 0.43, 0.02];
export const DEVICES: { key: string; share: number }[] = [
  { key: "MOBILE", share: 0.72 },
  { key: "DESKTOP", share: 0.24 },
  { key: "TABLET", share: 0.04 },
];
