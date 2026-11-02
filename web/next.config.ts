import type { NextConfig } from "next";

const backendURL = process.env.BPMN_MAPPER_BACKEND_URL ?? "http://127.0.0.1:8080";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  devIndicators: false,
  distDir: process.env.BPMN_MAPPER_TEST_BUILD === "1" ? ".next-test" : ".next",
	allowedDevOrigins: ["127.0.0.1"],
	experimental: {
		// Leave room beyond the Go server's 10-minute analysis deadline for its response.
		proxyTimeout: 660_000,
	},
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
