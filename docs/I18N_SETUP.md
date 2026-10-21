# Internationalization (i18n) Setup

This document describes the internationalization configuration for the AssetForge
frontend using next-intl with Next.js 16.

## Overview

The application uses [next-intl v4.14.9](https://next-intl-docs.vercel.app/),
which is fully compatible with Next.js 16.2.4. The configuration follows the
Next.js 16 App Router pattern without a separate middleware file.

## Supported Locales

The application supports four locales:

- **English (en)** - Default locale
- **Spanish (es)**
- **Chinese (zh)**
- **French (fr)**

## Configuration

### Core Configuration (`frontend/src/i18n/i18n.ts`)

```typescript
import { getRequestConfig } from 'next-intl/server';
import { cookies } from 'next/headers';

export const locales = ['en', 'es', 'zh', 'fr'] as const;
export type Locale = typeof locales[number];
export const defaultLocale: Locale = 'en';
export const LOCALE_COOKIE = 'NEXT_LOCALE';

export default getRequestConfig(async () => {
  const cookieStore = await cookies();
  const cookieLocale = cookieStore.get(LOCALE_COOKIE)?.value;
  const locale = isLocale(cookieLocale) ? cookieLocale : defaultLocale;

  return {
    locale,
    messages: (await import(`./locales/${locale}.json`)).default,
  };
});
```

This configuration:
- Uses server-side locale detection via cookies
- Validates locale values for security (prevents arbitrary file imports)
- Dynamically imports message catalogs
- Falls back to English if cookie is invalid or missing

### Next.js Configuration (`frontend/next.config.ts`)

The application configures next-intl via webpack alias instead of the plugin:

```typescript
turbopack: {
  resolveAlias: {
    "next-intl/config": "./src/i18n/i18n.ts",
  },
},
webpack(config) {
  config.resolve.alias["next-intl/config"] = path.resolve(
    process.cwd(),
    "src/i18n/i18n.ts"
  );
  return config;
}
```

This approach:
- Avoids loading the optional message-extraction plugin
- Uses runtime JSON catalogs (no compile-time extraction)
- Works with both webpack and Turbopack

### Root Layout (`frontend/src/app/layout.tsx`)

```tsx
import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages } from 'next-intl/server';

export default async function RootLayout({ children }) {
  const locale = await getLocale();
  const messages = await getMessages();

  return (
    <html lang={locale}>
      <body>
        <NextIntlClientProvider messages={messages}>
          {children}
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
```

## Message Files

Message catalogs live in `frontend/src/i18n/locales/`:

```
frontend/src/i18n/locales/
├── en.json
├── es.json
├── zh.json
└── fr.json
```

### Message Format

```json
{
  "common": {
    "welcome": "Welcome to AssetForge",
    "login": "Log In",
    "logout": "Log Out"
  },
  "portfolio": {
    "title": "Portfolio",
    "totalValue": "Total Value"
  }
}
```

Use nested keys for organization. Access in components:

```tsx
import { useTranslations } from 'next-intl';

function Portfolio() {
  const t = useTranslations('portfolio');
  return <h1>{t('title')}</h1>;
}
```

## Adding a New Locale

1. **Add locale to configuration** (`frontend/src/i18n/i18n.ts`):
   ```typescript
   export const locales = ['en', 'es', 'zh', 'fr', 'de'] as const;
   ```

2. **Create message file** (`frontend/src/i18n/locales/de.json`):
   ```json
   {
     "common": {
       "welcome": "Willkommen bei AssetForge"
     }
   }
   ```

3. **Test the locale**:
   - Set the `NEXT_LOCALE` cookie to the new locale code
   - Verify all pages load without errors
   - Check that messages display correctly

## Locale Switching

Users switch locales via `LanguageSelector` component, which calls the
`setLocale` server action:

```typescript
// frontend/src/actions/locale.ts
'use server';

import { cookies } from 'next/headers';
import { LOCALE_COOKIE } from '@/i18n/i18n';

export async function setLocale(locale: string) {
  const cookieStore = await cookies();
  cookieStore.set(LOCALE_COOKIE, locale, {
    httpOnly: false,
    sameSite: 'lax',
    secure: process.env.NODE_ENV === 'production',
  });
}
```

The cookie is read on the next request to determine the active locale.

## No Middleware Pattern

This setup intentionally does **not** use a separate `middleware.ts` file. With
Next.js 16, the locale is resolved server-side in the root layout and i18n
configuration, not in middleware. This pattern:

- Reduces middleware overhead
- Works with Next.js 16 caching and streaming
- Maintains full i18n functionality

If your application requires locale-based routing (e.g., `/en/about`,
`/es/about`), you would need to add middleware. The current configuration uses
cookie-based locale switching without URL prefixes.

## Formatting

### Numbers and Dates

Use the `useFormatter` hook for locale-aware formatting:

```tsx
import { useFormatter } from 'next-intl';

function AssetPrice({ value }: { value: number }) {
  const format = useFormatter();
  
  return (
    <div>
      <p>{format.number(value, { style: 'currency', currency: 'USD' })}</p>
      <p>{format.dateTime(new Date(), { dateStyle: 'long' })}</p>
    </div>
  );
}
```

## Compatibility

### Version Matrix

| Package     | Version  | Compatibility                  |
|-------------|----------|--------------------------------|
| Next.js     | 16.2.4   | Requires next-intl ≥4.4        |
| next-intl   | 4.14.9   | Supports Next.js 16.x          |
| React       | 19.2.4   | Supported                      |

### Upgrade Notes

**v4.14.9** (Issue #436): Verified compatibility with Next.js 16.2.4. No
configuration changes required. The current setup correctly uses:
- `getRequestConfig` from `next-intl/server`
- Server-side locale detection via cookies
- Dynamic message imports
- No separate middleware (valid Next.js 16 pattern)

## Testing

Verify i18n functionality:

1. **All locales load**:
   ```bash
   # In browser console, test each locale
   document.cookie = 'NEXT_LOCALE=es; path=/';
   location.reload();
   ```

2. **No missing messages**:
   Check browser console for `Missing message` warnings

3. **Formatting works**:
   Verify numbers and dates render in locale-specific format

## References

- [next-intl Documentation](https://next-intl-docs.vercel.app/)
- [Next.js 16 Internationalization](https://nextjs.org/docs/app/building-your-application/routing/internationalization)
- [next-intl Examples](https://github.com/amannn/next-intl/tree/main/examples)
