"use client"

import { ErrorFallback, type RouteErrorProps } from "@/components/errors/ErrorFallback"

/**
 * App-level error boundary. Catches errors thrown while rendering any route
 * segment below the root layout and renders a friendly, recoverable fallback
 * instead of crashing the whole app.
 */
export default function AppError({
  error,
  reset,
  retry,
  unstable_retry,
}: RouteErrorProps) {
  return (
    <ErrorFallback
      error={error}
      reset={reset}
      retry={retry}
      unstable_retry={unstable_retry}
    />
  )
}
