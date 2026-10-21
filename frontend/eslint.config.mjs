import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";
import requireUseServer from "./eslint-rules/require-use-server.js";

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,
  // Server Actions live in src/actions and must be 'use server' modules (#423).
  {
    files: ["src/actions/**/*.{ts,tsx,js,jsx}"],
    plugins: {
      local: { rules: { "require-use-server": requireUseServer } },
    },
    rules: {
      "local/require-use-server": "error",
    },
  },
  // Override default ignores of eslint-config-next.
  globalIgnores([
    // Default ignores of eslint-config-next:
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
  ]),
]);

export default eslintConfig;
