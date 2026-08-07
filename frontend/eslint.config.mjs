import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTypeScript from "eslint-config-next/typescript";
import storybook from "eslint-plugin-storybook";

export default defineConfig([
  ...nextVitals,
  ...nextTypeScript,
  ...storybook.configs["flat/recommended"],
  globalIgnores([
    ".next/**",
    "coverage/**",
    "playwright-report/**",
    "public/mockServiceWorker.js",
    "storybook-static/**",
    "test-results/**",
  ]),
]);
