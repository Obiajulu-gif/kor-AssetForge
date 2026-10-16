"use client"

import { ErrorFallback, type RouteErrorProps } from "@/components/errors/ErrorFallback"
import "./globals.css"

/**
 * Global error boundary. Replaces the root layout when an error is thrown in
 * the root layout itself, so it must render its own <html> and <body> tags
 * along with the global styles required to display the fallback.
 */
export default function GlobalError({
  error,
  reset,
  retry,
  unstable_retry,
}: RouteErrorProps) {
  return (
    <html lang="en">
      <body className="min-h-full antialiased">
        <ErrorFallback
          error={error}
          reset={reset}
          retry={retry}
          unstable_retry={unstable_retry}
          title="Application error"
          description="A critical error occurred and the app couldn't recover. Reload the page to continue."
        />
      </body>
    </html>
  )
}
