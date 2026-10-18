'use server';

import { cookies } from 'next/headers';
import { isLocale, LOCALE_COOKIE } from '@/i18n/i18n';

const ONE_YEAR_SECONDS = 60 * 60 * 24 * 365;

/**
 * Persists the user's language preference.
 *
 * Setting a cookie inside a Server Action makes Next.js re-render the current
 * page and layouts in the same round-trip, so callers do not need to call
 * router.refresh() afterwards.
 */
export async function setLocale(locale: string): Promise<void> {
  if (!isLocale(locale)) {
    throw new Error(`Unsupported locale: ${locale}`);
  }

  const cookieStore = await cookies();
  cookieStore.set(LOCALE_COOKIE, locale, {
    path: '/',
    maxAge: ONE_YEAR_SECONDS,
    sameSite: 'lax',
  });
}
