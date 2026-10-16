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
import { ErrorFallback } from "./ErrorFallback"

describe("ErrorFallback", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders a user-friendly message and reports the error", () => {
    render(<ErrorFallback error={new Error("boom")} reset={() => {}} />)

    expect(screen.getByText("Something went wrong")).toBeInTheDocument()
    expect(reportError).toHaveBeenCalledTimes(1)
  })

  it("offers a go-home recovery link", () => {
    render(<ErrorFallback error={new Error("boom")} reset={() => {}} />)

    expect(screen.getByRole("link", { name: /go home/i })).toHaveAttribute(
      "href",
      "/",
    )
  })

  it("invokes retry when the retry callback is provided", async () => {
    const retry = vi.fn()
    render(<ErrorFallback error={new Error("boom")} retry={retry} />)

    await userEvent.click(screen.getByRole("button", { name: /try again/i }))

    expect(retry).toHaveBeenCalledTimes(1)
  })

  it("falls back to reset when retry/unstable_retry are absent", async () => {
    const reset = vi.fn()
    render(<ErrorFallback error={new Error("boom")} reset={reset} />)

    await userEvent.click(screen.getByRole("button", { name: /try again/i }))

    expect(reset).toHaveBeenCalledTimes(1)
  })

  it("prefers retry over unstable_retry and reset", async () => {
    const retry = vi.fn()
    const unstableRetry = vi.fn()
    const reset = vi.fn()
    render(
      <ErrorFallback
        error={new Error("boom")}
        retry={retry}
        unstable_retry={unstableRetry}
        reset={reset}
      />,
    )

    await userEvent.click(screen.getByRole("button", { name: /try again/i }))

    expect(retry).toHaveBeenCalledTimes(1)
    expect(unstableRetry).not.toHaveBeenCalled()
    expect(reset).not.toHaveBeenCalled()
  })

  it("hides the retry button when no recovery handler is provided", () => {
    render(<ErrorFallback error={new Error("boom")} />)

    expect(
      screen.queryByRole("button", { name: /try again/i }),
    ).not.toBeInTheDocument()
  })

  it("tolerates non-Error thrown values", () => {
    render(<ErrorFallback error="plain string" reset={() => {}} />)

    expect(screen.getByText("Something went wrong")).toBeInTheDocument()
    expect(reportError).toHaveBeenCalledWith("plain string")
  })

  it("supports custom title and description", () => {
    render(
      <ErrorFallback
        error={new Error("boom")}
        reset={() => {}}
        title="Payments failed to load"
        description="Please come back later."
      />,
    )

    expect(screen.getByText("Payments failed to load")).toBeInTheDocument()
    expect(screen.getByText("Please come back later.")).toBeInTheDocument()
  })
})
