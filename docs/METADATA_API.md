# Metadata API

How SEO and social metadata are defined in the Next.js 16 App Router
frontend (issue #422).

## Audit summary (Next.js 16.2)

| File | Status |
| --- | --- |
| `src/app/layout.tsx` | Typed `Metadata` + separate `Viewport` export (`themeColor` lives in `viewport`, as required since Next.js 14). Added `metadataBase`, a title template, Open Graph, Twitter, and keywords. |
| `src/app/admin/layout.tsx` | Untyped object → typed `Metadata`. Title now goes through the root template. Marked `noindex, nofollow`. |
| `src/app/manifest.ts` | Already uses `MetadataRoute.Manifest`; no change needed. |
| `src/app/**/page.tsx` | All pages are Client Components (`'use client'`), which **cannot** export `metadata`. They inherit layout metadata; see "Per-route metadata" below. |

No route uses `generateMetadata` with `params`. If one is added, `params` is a
`Promise` in Next.js 16 and must be awaited.

## Root layout

```ts
export const metadata: Metadata = {
  metadataBase: new URL(process.env.NEXT_PUBLIC_SITE_URL || 'http://localhost:3000'),
  title: { default: 'kor-AssetForge', template: '%s | kor-AssetForge' },
  description: '…',
  openGraph: { type: 'website', siteName: 'kor-AssetForge', url: '/', images: [{ url: '/icon-512.png', width: 512, height: 512 }] },
  twitter: { card: 'summary', images: ['/icon-512.png'] },
};

export const viewport: Viewport = {
  themeColor: [
    { media: '(prefers-color-scheme: light)', color: '#ffffff' },
    { media: '(prefers-color-scheme: dark)', color: '#0a0a0a' },
  ],
};
```

- **`metadataBase`** turns relative Open Graph/Twitter URLs into absolute ones,
  which crawlers require. Set `NEXT_PUBLIC_SITE_URL` to the public origin in
  each deployment.
- **Title template:** child segments set `title: 'Admin Panel'` and render
  `Admin Panel | kor-AssetForge`.
- **`viewport` vs `metadata`:** `themeColor`, `colorScheme`, and `viewport`
  must be in the `viewport` export (or `generateViewport`). Putting them in
  `metadata` triggers a deprecation warning.

## Per-route metadata

`metadata` and `generateMetadata` are only supported in **Server Components**.
To title a route whose `page.tsx` is a Client Component, add a server
`layout.tsx` to that segment:

```tsx
// src/app/staking/layout.tsx
import type { Metadata } from 'next';
import type { ReactNode } from 'react';

export const metadata: Metadata = {
  title: 'Staking',
  description: 'Stake tokenized assets and earn rewards',
};

export default function StakingLayout({ children }: { children: ReactNode }) {
  return children;
}
```

For dynamic routes, use `generateMetadata` and await `params`:

```tsx
// src/app/assets/[id]/layout.tsx
import type { Metadata } from 'next';

export async function generateMetadata(
  { params }: { params: Promise<{ id: string }> },
): Promise<Metadata> {
  const { id } = await params;
  return { title: `Asset ${id}`, openGraph: { url: `/assets/${id}` } };
}
```

## Verifying

```bash
cd frontend
npm run build            # must finish with no metadata warnings
npm run start
curl -s http://localhost:3000 | grep -oE '<(title|meta (name|property)="(og|twitter|description)[^"]*")[^>]*>'
```
