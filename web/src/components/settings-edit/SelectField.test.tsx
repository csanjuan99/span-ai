import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { SelectField } from "./SelectField"

describe("SelectField", () => {
  it("stays a labelled native select and reports the picked value", () => {
    const onChange = vi.fn()
    render(
      <SelectField
        label="Motion"
        value="system"
        options={[
          { value: "system", label: "Follow system" },
          { value: "reduced", label: "Reduce motion" },
        ]}
        onChange={onChange}
      />
    )
    fireEvent.change(screen.getByRole("combobox", { name: "Motion" }), {
      target: { value: "reduced" },
    })
    expect(onChange).toHaveBeenCalledWith("reduced")
  })
})
