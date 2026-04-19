import { defineConfig } from "astro/config";
import tailwindcss from "@tailwindcss/vite";
import starlight from "@astrojs/starlight";

export default defineConfig({
  integrations: [
    starlight({
      title: "P&AI Bot",
      description: "Open-source AI learning agent that teaches students through chat.",
      social: [
        { icon: "github", label: "GitHub", href: "https://github.com/p-n-ai/pai-bot" },
      ],
      sidebar: [{ label: "Docs", autogenerate: { directory: "docs" } }],
      customCss: ["./src/styles/starlight.css"],
    }),
  ],
  server: { host: true },
  vite: {
    plugins: [tailwindcss()],
    server: { allowedHosts: [".trycloudflare.com"] },
  },
});
