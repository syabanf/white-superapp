"use client";

/**
 * Last-resort boundary for failures in the root layout itself. It replaces the
 * whole document, so it cannot rely on app CSS, fonts or providers.
 */
export default function GlobalError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <html lang="id">
      <body style={{ margin: 0, minHeight: "100dvh", display: "grid", placeItems: "center", fontFamily: "system-ui, sans-serif", background: "#fff", color: "#0a0a0a" }}>
        <div style={{ textAlign: "center", padding: 24, maxWidth: 420 }}>
          <h1 style={{ fontSize: 20, margin: 0 }}>Terjadi kesalahan</h1>
          <p style={{ fontSize: 14, color: "#5f6368", lineHeight: 1.6 }}>
            Aplikasi gagal dimuat{error.digest ? ` (kode ${error.digest})` : ""}. Coba muat ulang.
          </p>
          <button
            type="button"
            onClick={reset}
            style={{ marginTop: 8, height: 36, padding: "0 16px", borderRadius: 999, border: "1px solid #d4d4d8", background: "#fff", fontSize: 14, cursor: "pointer" }}
          >
            Coba lagi
          </button>
        </div>
      </body>
    </html>
  );
}
