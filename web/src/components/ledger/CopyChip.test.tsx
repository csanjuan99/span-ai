import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { CopyChip } from "@/components/ledger/CopyChip"

afterEach(cleanup)

describe("CopyChip", () => {
  it("copies the full value and says so, with the icon before the text", async () => {
    const writeText = vi.fn(() => Promise.resolve())
    vi.stubGlobal("navigator", { clipboard: { writeText } })
    render(
      <CopyChip
        label="digest"
        value="sha256:abc"
        icon={<svg data-testid="copy-icon" />}
      >
        Copy digest
      </CopyChip>
    )
    const button = screen.getByRole("button", {
      name: "Copy digest sha256:abc",
    })
    expect(button.firstElementChild?.getAttribute("data-testid")).toBe(
      "copy-icon"
    )
    fireEvent.click(button)
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("sha256:abc"))
    expect(await screen.findByLabelText("copied")).toBeTruthy()
  })
})
