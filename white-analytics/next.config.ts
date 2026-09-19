import type { NextConfig } from "next";

const dev = process.env.NODE_ENV !== "production";

/**
 * CSP notes: Next injects inline bootstrap scripts and Tailwind/Radix write inline
 * styles, so both need 'unsafe-inline' until the app moves to nonces. Images and
 * media allow https because avatars, post thumbnails and library assets come from
 * provider CDNs. Dev adds eval + websockets for Turbopack HMR.
 */
const csp = [
  "default-src 'self'",
  `script-src 'self' 'unsafe-inline'${dev ? " 'unsafe-eval'" : ""}`,
  "style-src 'self' 'unsafe-inline'",
  "img-src 'self' data: blob: https:",
  "media-src 'self' blob: https:",
  "font-src 'self' data:",
  `connect-src 'self'${dev ? " ws: wss:" : ""}`,
  "frame-ancestors 'none'",
  "base-uri 'self'",
  "form-action 'self'",
  "object-src 'none'",
].join("; ");

const securityHeaders = [
  { key: "Content-Security-Policy", value: csp },
  { key: "X-Frame-Options", value: "DENY" },
  { key: "X-Content-Type-Options", value: "nosniff" },
  { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
  { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=(), payment=()" },
  { key: "Strict-Transport-Security", value: "max-age=63072000; includeSubDomains" },
];

const nextConfig: NextConfig = {
  // Dev only: the in-app browser reaches the dev server as 127.0.0.1 (not localhost).
  allowedDevOrigins: ["127.0.0.1"],
  experimental: {
    // Media uploads (publishing module) go through server actions.
    serverActions: { bodySizeLimit: "30mb" },
  },
  async headers() {
    return [{ source: "/(.*)", headers: securityHeaders }];
  },
};

export default nextConfig;
