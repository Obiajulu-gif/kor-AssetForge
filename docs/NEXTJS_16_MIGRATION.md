# Next.js 16 / React 19 Migration Notes

Configuration decisions for the frontend on Next.js 16.2 and React 19.2
(issues #420, #421). See also [SERVER_ACTIONS.md](SERVER_ACTIONS.md) and
[METADATA_API.md](METADATA_API.md).

## JSX transform (#420)

`frontend/tsconfig.json` uses `"jsx": "react-jsx"`. That's the correct value for
Next.js 16. Next.js enforces it: `next build`/`next dev` rewrite any other
value with the reason *"next.js uses the React automatic runtime"*. The old
`"preserve"` setting from Next.js ≤15 templates no longer applies.

With the automatic runtime:

- **No `import React from 'react'` is needed for JSX.** Import only what you use:
  ```tsx
  import type { ReactNode } from 'react';
  import { useState } from 'react';
  ```
- Existing `import React` / `import * as React` lines are harmless when the file
  uses `React.*` APIs or types. Remove them when you touch a file and they're
  unused. The root and admin layouts now use `import type { ReactNode }`.

## React Compiler (#420)

React Compiler 1.0 is stable in Next.js 16 through the top-level
`reactCompiler` option (no longer under `experimental`). It's enabled in
**annotation mode**:

```ts
// frontend/next.config.ts
reactCompiler: {
  compilationMode: 'annotation',
},
```

Only components and hooks that opt in are compiled:

```tsx
export function PriceTicker({ prices }: Props) {
  'use memo';
  // ...
}
```

We don't use full mode (`reactCompiler: true`) yet because ESLint currently
reports `react-hooks/purity`, `react-hooks/immutability`, and
`react-hooks/set-state-in-effect` violations in existing components. The
compiler assumes the Rules of React, so compiling those components could
change their behavior. To migrate:

1. Fix the `react-hooks/*` compiler-rule lint errors in a component.
2. Add `'use memo'` to it and verify.
3. Once the codebase is clean, switch to `reactCompiler: true` and use
   `'use no memo'` for any exceptions.

The compiler runs through `babel-plugin-react-compiler` (a dev dependency).
Next.js applies it only to files containing JSX or hooks, so the build-time
cost is small.

## TypeScript types (#421)

Next.js 16 generates its types instead of shipping them in the repo:

| Generated file | Purpose |
| --- | --- |
| `next-env.d.ts` | References `next` and `next/image-types/global` and imports `.next/types/routes.d.ts`. |
| `.next/types/routes.d.ts` | Global `PageProps<'/route'>`, `LayoutProps<'/route'>`, and `RouteContext` helpers. |
| `.next/types/validator.ts` | Checks that every page/layout export matches Next.js' expected types. |
| `.next/dev/types/**` | The same, written by `next dev` (Next.js 16 isolates dev output in `.next/dev`). |

They're produced by `next dev`, `next build`, or `next typegen`, and they're
ignored by `frontend/.gitignore` (`/.next/`, `next-env.d.ts`, `*.tsbuildinfo`).
`tsconfig.json` already includes `next-env.d.ts`, `.next/types/**/*.ts`, and
`.next/dev/types/**/*.ts`.

Type-check without a full build:

```bash
cd frontend
npm run typecheck   # next typegen && tsc --noEmit
```

CI runs this before the unit tests. Project-specific declarations, such as typed
`process.env.NEXT_PUBLIC_*` variables, go in `frontend/src/types/next.d.ts`.

Route props in Next.js 16 are async. With the generated helpers:

```tsx
export default async function Page(props: PageProps<'/assets/[id]'>) {
  const { id } = await props.params;
}
```

Client Components keep using `useParams()` from `next/navigation`.
