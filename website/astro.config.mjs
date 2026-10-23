import { defineConfig } from "astro/config";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  site: "https://the-sumeet.github.io",
  base: "/dotmd",
  vite: {
    plugins: [tailwindcss()],
  },
});
