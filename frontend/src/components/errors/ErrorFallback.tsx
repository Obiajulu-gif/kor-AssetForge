"use client"

import { useEffect } from "react"
import Link from "next/link"
import { AlertTriangle, RefreshCw, Home } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { reportError } from "@/lib/error-reporting"

/**
 * Props passed by the Next.js App Router to `error.tsx` / `global-error.tsx`
 * boundaries.
 *
 * Next.js renamed the recover callback over time (`reset` -> `unstable_retry`
 * (16.2) -> `retry` (16.3)). We accept every variant so the same component
 * works across versions and degrade gracefully when none is provided.
 */
export interface RouteErrorProps {
  error: unknown
  reset?: () => void
  retry?: () => void
  unstable_retry?: () => void
}

interface ErrorFallbackProps extends RouteErrorProps {
  title?: string
  description?: string
}

function toError(error: unknown): Error {
  return error instanceof Error ? error : new Error(String(error))
}

/**
 * Shared user-friendly fallback rendered by route-level error boundaries.
 *
 * Reports the error to the monitoring pipeline, shows a friendly message, and
 * offers recovery (retry) and escape (go home) actions.
 */
export function ErrorFallback({
  error,
  reset,
  retry,
  unstable_retry,
  title = "Something went wrong",
  description = "An unexpected error occurred while loading this page. Please try again.",
}: ErrorFallbackProps) {
  const recover = retry ?? unstable_retry ?? reset
  const isDev = process.env.NODE_ENV === "development"

  useEffect(() => {
    reportError(error)
  }, [error])

  const err = toError(error)

  return (
    <div className="flex min-h-[400px] items-center justify-center p-8">
      <Card className="w-full max-w-lg">
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10">
              <AlertTriangle className="h-6 w-6 text-destructive" />
            </div>
            <div>
              <CardTitle>{title}</CardTitle>
              <CardDescription>{description}</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {isDev && (
            <div className="rounded-lg bg-muted p-4">
              <p className="mb-2 font-mono text-sm font-semibold text-destructive">
                {err.name}: {err.message}
              </p>
              {err.stack && (
                <pre className="max-h-40 overflow-auto text-xs text-muted-foreground">
                  {err.stack}
                </pre>
              )}
            </div>
          )}
          <div className="flex gap-3">
            {recover && (
              <Button onClick={() => recover()}>
                <RefreshCw className="mr-2 h-4 w-4" />
                Try Again
              </Button>
            )}
            <Link href="/">
              <Button variant="outline">
                <Home className="mr-2 h-4 w-4" />
                Go Home
              </Button>
            </Link>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
