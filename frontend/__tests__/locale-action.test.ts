/**
 * @jest-environment node
 */
const setCookie = jest.fn();

jest.mock("next/headers", () => ({
  cookies: jest.fn(async () => ({ set: setCookie, get: jest.fn() })),
}));

jest.mock("next-intl/server", () => ({
  getRequestConfig: (fn: unknown) => fn,
}));

import { setLocale } from "@/actions/locale";
import { isLocale, LOCALE_COOKIE } from "@/i18n/i18n";

describe("isLocale", () => {
  it("accepts supported locales", () => {
    expect(["en", "es", "zh", "fr"].every(isLocale)).toBe(true);
  });

  it("rejects unknown or malformed values", () => {
    for (const value of ["de", "", "EN", "../en", "en.json", undefined, null, 1]) {
      expect(isLocale(value)).toBe(false);
    }
  });
});

describe("setLocale server action", () => {
  beforeEach(() => setCookie.mockClear());

  it("persists a supported locale in the locale cookie", async () => {
    await setLocale("fr");

    expect(setCookie).toHaveBeenCalledWith(LOCALE_COOKIE, "fr", {
      path: "/",
      maxAge: 60 * 60 * 24 * 365,
      sameSite: "lax",
    });
  });

  it("rejects unsupported locales without touching cookies", async () => {
    await expect(setLocale("../../etc/passwd")).rejects.toThrow("Unsupported locale");
    expect(setCookie).not.toHaveBeenCalled();
  });
});
