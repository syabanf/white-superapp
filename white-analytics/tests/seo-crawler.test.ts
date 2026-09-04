import { describe, it, expect } from "vitest";
import { analyzeHtml } from "@/lib/providers/crawler/analyze";
import { isAllowed, parseRobots, parseSitemap } from "@/lib/providers/crawler/robots";

const BASE = "https://contoh.id/produk";

function codes(issues: { code: string }[]): string[] {
  return issues.map((i) => i.code);
}

const GOOD_HTML = `<!doctype html><html><head>
  <title>Kopi Nusantara — Biji Kopi Arabika Gayo Premium</title>
  <meta name="description" content="Beli biji kopi arabika Gayo premium, sangrai fresh setiap minggu, gratis ongkir Jabodetabek.">
  <link rel="canonical" href="https://contoh.id/produk">
  <link rel="alternate" hreflang="en" href="https://contoh.id/en/products">
  <script type="application/ld+json">{"@type":"Product"}</script>
</head><body>
  <h1>Biji Kopi Arabika Gayo</h1>
  <p>${"kopi arabika nusantara terbaik dari dataran tinggi gayo ".repeat(60)}</p>
  <img src="/a.jpg" alt="Biji kopi">
  <img src="/b.jpg" alt="">
  <a href="/menu">Menu</a>
  <a href="/lokasi#peta">Lokasi</a>
  <a href="https://instagram.com/kopi">IG</a>
  <a href="mailto:halo@contoh.id">Email</a>
  <a href="/produk">Self</a>
</body></html>`;

describe("crawler analyzeHtml", () => {
  it("extracts fields from a healthy page and reports no issues", () => {
    const out = analyzeHtml({ url: BASE, html: GOOD_HTML, statusCode: 200, loadMs: 320 });
    expect(out.page.title).toContain("Kopi Nusantara");
    expect(out.page.metaDescription).toContain("arabika");
    expect(out.page.h1Count).toBe(1);
    expect(out.page.wordCount).toBeGreaterThan(300);
    expect(out.page.canonical).toBe("https://contoh.id/produk");
    expect(out.page.indexable).toBe(true);
    expect(out.page.imagesTotal).toBe(2);
    expect(out.page.imagesMissingAlt).toBe(0); // alt="" counts as present (decorative)
    expect(out.page.hasJsonLd).toBe(true);
    expect(out.page.hasHreflang).toBe(true);
    expect(out.page.internalLinks).toBe(2); // /menu + /lokasi (hash stripped, self-link ignored)
    expect(out.page.externalLinks).toBe(1); // instagram only (mailto skipped)
    expect(out.internalUrls).toContain("https://contoh.id/menu");
    expect(out.internalUrls).toContain("https://contoh.id/lokasi");
    expect(out.issues).toHaveLength(0);
  });

  it("flags a broken page with the full issue set", () => {
    const html = `<html><head></head><body>
      <h1>Satu</h1><h1>Dua</h1>
      <p>Konten pendek.</p>
      <img src="/x.jpg"><img src="/y.jpg">
      <img src="http://insecure.com/z.jpg" alt="z">
      <meta name="robots" content="noindex, nofollow">
    </body></html>`;
    const out = analyzeHtml({ url: "https://contoh.id/rusak", html, statusCode: 200, loadMs: 2100 });
    const c = codes(out.issues);
    expect(c).toContain("MISSING_TITLE");
    expect(c).toContain("MISSING_META_DESCRIPTION");
    expect(c).toContain("MULTIPLE_H1");
    expect(c).toContain("IMG_MISSING_ALT");
    expect(c).toContain("MISSING_CANONICAL");
    expect(c).toContain("NOINDEX");
    expect(c).toContain("THIN_CONTENT");
    expect(c).toContain("SLOW_RESPONSE");
    expect(c).toContain("NO_STRUCTURED_DATA");
    expect(c).toContain("MIXED_CONTENT");
    expect(out.page.indexable).toBe(false);
    expect(out.page.imagesMissingAlt).toBe(2);
    const alt = out.issues.find((i) => i.code === "IMG_MISSING_ALT");
    expect(alt?.message).toContain("2 gambar");
  });

  it("checks title length boundaries", () => {
    const make = (title: string) =>
      analyzeHtml({ url: "https://contoh.id/", html: `<html><head><title>${title}</title></head><body><h1>x</h1></body></html>`, statusCode: 200 });
    expect(codes(make("Pendek").issues)).toContain("TITLE_TOO_SHORT");
    expect(codes(make("T".repeat(61)).issues)).toContain("TITLE_TOO_LONG");
    const okTitle = make("Judul halaman yang pas antara tiga puluh dan enam puluh");
    expect(codes(okTitle.issues)).not.toContain("TITLE_TOO_SHORT");
    expect(codes(okTitle.issues)).not.toContain("TITLE_TOO_LONG");
  });

  it("flags redirect chains and X-Robots-Tag noindex", () => {
    const out = analyzeHtml({
      url: "https://contoh.id/redirected",
      html: "<html><head><title>Halaman tujuan redirect yang valid sekali</title></head><body><h1>x</h1></body></html>",
      statusCode: 200,
      redirectHops: 2,
      xRobotsTag: "noindex",
    });
    expect(codes(out.issues)).toContain("REDIRECT_CHAIN");
    expect(codes(out.issues)).toContain("NOINDEX");
    expect(out.page.indexable).toBe(false);
  });

  it("skips content checks for error responses", () => {
    const out = analyzeHtml({ url: "https://contoh.id/404", html: "<html><body>Tidak ditemukan</body></html>", statusCode: 404 });
    expect(out.issues).toHaveLength(0);
    expect(out.page.indexable).toBe(false);
  });
});

describe("crawler robots", () => {
  const robots = `
# komentar
User-agent: Googlebot
Disallow: /private/

User-agent: WHITEAnalyticsBot
Disallow: /rahasia/
Allow: /rahasia/publik

User-agent: *
Disallow: /admin/

Sitemap: https://contoh.id/sitemap.xml
`;

  it("selects the specific UA group and honours allow overrides", () => {
    const rules = parseRobots(robots, "whiteanalyticsbot");
    expect(rules.sitemaps).toEqual(["https://contoh.id/sitemap.xml"]);
    expect(isAllowed(rules, "/rahasia/data")).toBe(false);
    expect(isAllowed(rules, "/rahasia/publik/laporan")).toBe(true);
    // the * group does not apply because a specific group matched
    expect(isAllowed(rules, "/admin/panel")).toBe(true);
    expect(isAllowed(rules, "/menu")).toBe(true);
  });

  it("falls back to the * group and supports wildcards", () => {
    const rules = parseRobots("User-agent: *\nDisallow: /*.pdf$\nDisallow: /cart", "whiteanalyticsbot");
    expect(isAllowed(rules, "/files/brosur.pdf")).toBe(false);
    expect(isAllowed(rules, "/files/brosur.pdf?x=1")).toBe(true); // $ anchors the end
    expect(isAllowed(rules, "/cart/checkout")).toBe(false);
    expect(isAllowed(rules, "/menu")).toBe(true);
  });

  it("parses url sets and sitemap indexes", () => {
    const urlset = `<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
      <url><loc>https://contoh.id/</loc></url><url><loc>https://contoh.id/menu</loc></url></urlset>`;
    expect(parseSitemap(urlset).urls).toEqual(["https://contoh.id/", "https://contoh.id/menu"]);
    const index = `<?xml version="1.0"?><sitemapindex><sitemap><loc>https://contoh.id/sitemap-1.xml</loc></sitemap></sitemapindex>`;
    expect(parseSitemap(index).sitemaps).toEqual(["https://contoh.id/sitemap-1.xml"]);
  });
});
