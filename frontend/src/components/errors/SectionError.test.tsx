import type { ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"

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

import { SectionError } from "./SectionError"

describe("SectionError", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("uses section-specific messaging", () => {
    render(
      <SectionError section="Dashboard" error={new Error("boom")} reset={() => {}} />,
    )

    expect(screen.getByText("Dashboard failed to load")).toBeInTheDocument()
    expect(
      screen.getByText(/couldn't load the Dashboard section/i),
    ).toBeInTheDocument()
  })

  it("still provides recovery actions", () => {
    render(
      <SectionError section="Trading" error={new Error("boom")} reset={() => {}} />,
    )

    expect(screen.getByRole("button", { name: /try again/i })).toBeInTheDocument()
    expect(screen.getByRole("link", { name: /go home/i })).toBeInTheDocument()
  })
})
