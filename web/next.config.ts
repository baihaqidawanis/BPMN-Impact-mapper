import type { NextConfig } from "next";

const backendURL = process.env.BPMN_MAPPER_BACKEND_URL ?? "http://127.0.0.1:8080";

const nextConfig: NextConfig = {
  reactStrictMode: true,
	allowedDevOrigins: ["127.0.0.1"],
  async rewrites() {
    return [
      {
        source: "/backend/:path*",
        destination: `${backendURL}/:path*`,
      },
    ];
  },
};

export default nextConfig;
