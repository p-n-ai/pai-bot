import { defineConfig } from "astro/config";
import tailwindcss from "@tailwindcss/vite";
import starlight from "@astrojs/starlight";

const site = process.env.SITE_URL ?? "https://p-n-ai.github.io/pai-bot";
const base = site ? new URL(site).pathname.replace(/\/$/, "") || "/" : "/";

export default defineConfig({
  site,
  base,
  integrations: [
    starlight({
      title: "P&AI Bot",
      description: "Open-source AI learning agent that teaches students through chat.",
      favicon: "/favicon.svg",
      logo: {
        src: "../docs/assets/pandai-logo.png",
        alt: "Pandai",
      },
      social: [
        { icon: "github", label: "GitHub", href: "https://github.com/p-n-ai/pai-bot" },
      ],
      sidebar: [
        { label: "Getting started", autogenerate: { directory: "getting-started" } },
        { label: "Features", autogenerate: { directory: "features" } },
        { label: "Guides", autogenerate: { directory: "guides" } },
        { label: "Deployment", autogenerate: { directory: "deployment" } },
      ],
      customCss: ["./src/styles/starlight.css"],
      lastUpdated: true,
    }),
  ],
  server: { host: true },
  vite: {
    plugins: [tailwindcss()],
    server: { allowedHosts: [".trycloudflare.com"] },
  },
});
