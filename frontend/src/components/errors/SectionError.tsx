"use client"

import { ErrorFallback, type RouteErrorProps } from "./ErrorFallback"

export interface SectionErrorProps extends RouteErrorProps {
  /** Human-readable name of the section, e.g. "Dashboard". */
  section: string
}

/**
 * Fallback used by major section boundaries (`app/<section>/error.tsx`). It
 * tailors the copy to the section while keeping recovery behaviour identical,
 * so a failure in one area never takes down the rest of the app.
 */
export function SectionError({ section, ...rest }: SectionErrorProps) {
  return (
    <ErrorFallback
      {...rest}
      title={`${section} failed to load`}
      description={`We couldn't load the ${section} section. This is usually temporary — try again or head back home.`}
    />
  )
}
