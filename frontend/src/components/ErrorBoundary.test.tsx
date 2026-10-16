import type { ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"

vi.mock("next/link", async () => {
  const React = await import("react")
  return {
    default: ({ href, children }: { href?: unknown; children?: ReactNode }) =>
      React.createElement(
        "a",
        { href: typeof href === "string" ? href : "#" },
        children,
      ),
  }
})

vi.mock("@/lib/error-reporting", () => ({ reportError: vi.fn() }))

import { reportError } from "@/lib/error-reporting"
import { ErrorBoundary } from "./ErrorBoundary"

function Boom(): ReactNode {
  throw new Error("kaboom")
}

describe("ErrorBoundary", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(console, "error").mockImplementation(() => {})
  })

  it("renders its children when there is no error", () => {
    render(
      <ErrorBoundary>
        <div>all good</div>
      </ErrorBoundary>,
    )

    expect(screen.getByText("all good")).toBeInTheDocument()
    expect(reportError).not.toHaveBeenCalled()
  })

  it("renders the fallback UI and reports the error when a child throws", () => {
    render(
      <ErrorBoundary>
        <Boom />
      </ErrorBoundary>,
    )

    expect(screen.getByText("Something went wrong")).toBeInTheDocument()
    expect(reportError).toHaveBeenCalledTimes(1)
  })

  it("supports a custom fallback", () => {
    render(
      <ErrorBoundary fallback={<div>custom fallback</div>}>
        <Boom />
      </ErrorBoundary>,
    )

    expect(screen.getByText("custom fallback")).toBeInTheDocument()
  })

  it("re-renders the children after the user retries", async () => {
    let shouldThrow = true

    function ThrowOnce(): ReactNode {
      if (shouldThrow) {
        throw new Error("once")
      }
      return <div>recovered</div>
    }

    render(
      <ErrorBoundary>
        <ThrowOnce />
      </ErrorBoundary>,
    )

    expect(screen.getByText("Something went wrong")).toBeInTheDocument()

    shouldThrow = false
    await userEvent.click(screen.getByRole("button", { name: /try again/i }))

    expect(screen.getByText("recovered")).toBeInTheDocument()
  })

  it("invokes the onError callback when provided", () => {
    const onError = vi.fn()

    render(
      <ErrorBoundary onError={onError}>
        <Boom />
      </ErrorBoundary>,
    )

    expect(onError).toHaveBeenCalledTimes(1)
  })
})
