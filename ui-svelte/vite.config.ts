import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    svelte(),
    tailwindcss(),
  ],
  base: "/ui/",
  build: {
    outDir: "../proxy/ui_dist",
    assetsDir: "assets",
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) {
            return undefined;
          }

          const normalized = id.replace(/\\/g, "/");
          if (normalized.includes("/@codemirror/") || normalized.includes("/codemirror/")) {
            return "vendor-codemirror";
          }
          if (normalized.includes("/katex/")) {
            return "vendor-katex";
          }
          if (normalized.includes("/highlight.js/")) {
            return "vendor-highlight";
          }
          if (normalized.includes("/lucide-svelte/")) {
            return "vendor-icons";
          }
          if (normalized.includes("/svelte-spa-router/")) {
            return "vendor-router";
          }
          return "vendor-shared";
        },
      },
    },
  },
  server: {
    proxy: {
      "/api": "http://localhost:8080", // Proxy API calls to Go backend during development
      "/logs": "http://localhost:8080",
      "/upstream": "http://localhost:8080",
      "/unload": "http://localhost:8080",
      "/v1": "http://localhost:8080",
    },
  },
});
