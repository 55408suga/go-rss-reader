import { defineConfig } from "vitest/config";

// Unit tests target pure TypeScript modules, so disable CSS handling and the
// project's Tailwind v4 PostCSS pipeline (which is not a valid PostCSS plugin
// outside next build and otherwise breaks Vitest's config loading).
export default defineConfig({
  css: { postcss: { plugins: [] } },
  test: {
    environment: "node",
    css: false,
    include: ["src/**/*.test.ts"],
  },
});
