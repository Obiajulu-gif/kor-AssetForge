import type { ReactNode } from "react";
import { render, screen } from "@testing-library/react";

jest.mock("next/link", () => {
  const React = require("react");
  return {
    __esModule: true,
    default: ({ href, children }: { href?: unknown; children?: ReactNode }) =>
      React.createElement(
        "a",
        { href: typeof href === "string" ? href : "#" },
        children,
      ),
  };
});

jest.mock("@/lib/error-reporting", () => ({ reportError: jest.fn() }));

import { SectionError } from "@/components/errors/SectionError";

describe("SectionError", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("uses section-specific messaging", () => {
    render(
      <SectionError section="Dashboard" error={new Error("boom")} reset={() => {}} />,
    );

    expect(screen.getByText("Dashboard failed to load")).toBeInTheDocument();
    expect(
      screen.getByText(/couldn't load the Dashboard section/i),
    ).toBeInTheDocument();
  });

  it("still provides recovery actions", () => {
    render(
      <SectionError section="Trading" error={new Error("boom")} reset={() => {}} />,
    );

    expect(screen.getByRole("button", { name: /try again/i })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /go home/i })).toBeInTheDocument();
  });
});
