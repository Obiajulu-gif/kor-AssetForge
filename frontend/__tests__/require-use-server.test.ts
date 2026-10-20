/**
 * @jest-environment node
 */
import { RuleTester } from "eslint";
import * as tsParser from "@typescript-eslint/parser";
import rule from "../eslint-rules/require-use-server";

RuleTester.describe = describe;
RuleTester.it = it;
RuleTester.itOnly = it.only;

const ruleTester = new RuleTester({
  languageOptions: { parser: tsParser, ecmaVersion: "latest", sourceType: "module" },
});

ruleTester.run("require-use-server", rule, {
  valid: [
    `'use server';\nexport async function save(data: FormData) {}`,
    `"use strict";\n"use server";\nexport const remove = async (id: string) => {};`,
    `'use server';\nexport default async function action() {}`,
    `'use server';\nexport type Result = { ok: boolean };\nexport async function run(): Promise<Result> { return { ok: true }; }`,
    `'use server';\nimport { cookies } from 'next/headers';\nconst helper = () => 1;\nexport async function read() { return helper(); }`,
  ],
  invalid: [
    {
      code: `export async function save() {}`,
      errors: [{ messageId: "missingDirective" }],
    },
    {
      code: `import x from 'y';\n'use server';\nexport async function save() { return x; }`,
      errors: [{ messageId: "missingDirective" }],
    },
    {
      code: `'use server';\nexport function save() {}`,
      errors: [{ messageId: "nonAsyncExport" }],
    },
    {
      code: `'use server';\nexport const LIMIT = 10;`,
      errors: [{ messageId: "nonAsyncExport" }],
    },
    {
      code: `'use server';\nexport const save = () => {};`,
      errors: [{ messageId: "nonAsyncExport" }],
    },
    {
      code: `'use server';\nasync function save() {}\nexport { save };`,
      errors: [{ messageId: "nonAsyncExport" }],
    },
    {
      code: `'use server';\nexport * from './other';`,
      errors: [{ messageId: "nonAsyncExport" }],
    },
    {
      code: `'use server';\nexport default { save: async () => {} };`,
      errors: [{ messageId: "nonAsyncExport" }],
    },
  ],
});
