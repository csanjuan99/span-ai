import { act, cleanup, render } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { useRowKeys } from "@/hooks/use-row-keys"

function Rows() {
  useRowKeys()
  return (
    <ul>
      <li>
        <a href="/a" data-row-link>
          A
        </a>
      </li>
      <li>
        <a href="/b" data-row-link>
          B
        </a>
      </li>
      <li>
        <a href="/c" data-row-link>
          C
        </a>
      </li>
      <li>
        <input aria-label="Search" />
      </li>
    </ul>
  )
}

function press(key: string, target: EventTarget = document.body) {
  act(() => {
    target.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true }))
  })
}

afterEach(cleanup)

describe("useRowKeys", () => {
  it("J moves to the next row and K to the previous one", () => {
    render(<Rows />)
    press("j")
    expect(document.activeElement?.textContent).toBe("A")
    press("j")
    expect(document.activeElement?.textContent).toBe("B")
    press("k")
    expect(document.activeElement?.textContent).toBe("A")
  })

  it("stops at the first and the last row", () => {
    render(<Rows />)
    press("k")
    expect(document.activeElement?.textContent).toBe("A")
    press("j")
    press("j")
    press("j")
    expect(document.activeElement?.textContent).toBe("C")
  })

  it("stays out of the way while the person types", () => {
    const { getByLabelText } = render(<Rows />)
    const field = getByLabelText("Search")
    field.focus()
    press("j", field)
    expect(document.activeElement).toBe(field)
  })
})
