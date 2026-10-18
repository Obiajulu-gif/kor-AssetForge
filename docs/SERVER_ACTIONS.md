# Server Actions

Conventions for Server Actions in the Next.js 16 frontend (issue #423).

## Where actions live

All Server Actions live in `frontend/src/actions/*.ts`. Each file is a
**`'use server'` module**: the directive is the first statement, so every
export is compiled as a Server Action. Client bundles only get an RPC
reference to the action, never its body or its server-only imports
(`next/headers`, secrets, and so on). That is what keeps actions tree-shaken
out of the client.

```ts
// frontend/src/actions/locale.ts
'use server';

import { cookies } from 'next/headers';
import { isLocale, LOCALE_COOKIE } from '@/i18n/i18n';

export async function setLocale(locale: string): Promise<void> {
  if (!isLocale(locale)) {
    throw new Error(`Unsupported locale: ${locale}`);
  }
  const cookieStore = await cookies(); // async-only in Next.js 16
  cookieStore.set(LOCALE_COOKIE, locale, { path: '/', maxAge: 31536000, sameSite: 'lax' });
}
```

## Calling an action from a Client Component

Import the action and call it inside a transition, or pass it to a form's
`action` prop:

```tsx
'use client';

import { useTransition } from 'react';
import { setLocale } from '@/actions/locale';

export function LocaleSwitch() {
  const [isPending, startTransition] = useTransition();
  return (
    <select
      disabled={isPending}
      onChange={(e) => startTransition(async () => { await setLocale(e.target.value); })}
    >
      {/* options */}
    </select>
  );
}
```

```tsx
// Form usage: the action receives FormData.
<form action={saveSettings}>
  <input name="displayName" />
  <button type="submit">Save</button>
</form>
```

Setting or deleting a cookie in an action makes Next.js re-render the current
page and its layouts in the same round-trip, so there's no need to call
`router.refresh()`. To refresh cached data too, call `revalidatePath` or
`revalidateTag` inside the action.

## Inline actions in Server Components

An action defined *inside* a Server Component needs `'use server'` as the
first statement of its function body:

```tsx
export default async function Page() {
  async function archive(formData: FormData) {
    'use server';
    // ...
  }
  return <form action={archive}>{/* ... */}</form>;
}
```

Prefer `src/actions/` over inline actions. A module can be imported by Client
Components and is covered by the lint rule below.

## Rules

1. **Directive first.** `'use server'` must be the first statement of the file
   (only other directives such as `'use strict'` may come before it).
2. **Async functions only.** A `'use server'` module may export only async
   functions (type exports are fine). Constants, helpers, and re-exports go in
   a separate module. For example, `isLocale` lives in `src/i18n/i18n.ts`, not
   in the action file.
3. **Treat arguments as untrusted input.** Actions are public HTTP endpoints.
   Validate every argument, and authenticate/authorize from cookies or headers
   rather than from tokens passed as arguments.
4. **Use async Request APIs.** `cookies()`, `headers()`, and `draftMode()` must
   be awaited in Next.js 16.

## Lint enforcement

`frontend/eslint-rules/require-use-server.js` is registered in
`frontend/eslint.config.mjs` (as `local/require-use-server`) for
`src/actions/**`. It reports:

- files that don't start with `'use server'`
- non-async exports, `export { x }` re-exports, and `export *`

The repository uses ESLint 9 flat config (`eslint.config.mjs`), not
`.eslintrc.json`, since Next.js 16 removed `next lint` and legacy configs.
Run it with:

```bash
cd frontend
npx eslint src/actions
```

Rule tests live in `frontend/__tests__/require-use-server.test.ts`.
