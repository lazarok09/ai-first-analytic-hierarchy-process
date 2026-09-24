import type { NextConfig } from "next";

/** Project-site path on GitHub Pages (repo name). */
const basePath = "/ai-first-analytic-hierarchy-process";

const nextConfig: NextConfig = {
  output: "export",
  images: { unoptimized: true },
  trailingSlash: true,
  basePath,
  assetPrefix: basePath,
};

export default nextConfig;
