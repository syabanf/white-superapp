import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Dev only: the in-app browser reaches the dev server as 127.0.0.1 (not localhost).
  allowedDevOrigins: ["127.0.0.1"],
  experimental: {
    // Media uploads (publishing module) go through server actions.
    serverActions: { bodySizeLimit: "30mb" },
  },
};

export default nextConfig;
