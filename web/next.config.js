/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "export",
  images: { unoptimized: true },
  trailingSlash: true,
  // WSL browsers often open 127.0.0.1. Without this, Next blocks the dev CSS and JS.
  allowedDevOrigins: ["127.0.0.1"],
};

module.exports = nextConfig;
