import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { PromptHeading } from "@/components/shell/PromptHeading"

afterEach(cleanup)

describe("PromptHeading", () => {
  it("is a level-two heading with the prompt before the text", () => {
    render(<PromptHeading>story</PromptHeading>)
    const heading = screen.getByRole("heading", { level: 2 })
    expect(heading.textContent).toBe("$ story")
  })

  it("renders a page title as a level-one heading", () => {
    render(<PromptHeading as="h1">shape a story</PromptHeading>)
    expect(screen.getByRole("heading", { level: 1 }).textContent).toBe(
      "$ shape a story"
    )
  })
})
