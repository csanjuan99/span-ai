"use client"

import { configDefaults } from "@/lib/api"

import { CommandsField, TextField, type EditProps } from "./fields"

/** The checks that prove the work, editable. Saved to span.yaml. */
export function ChecksFields({ props }: { props: EditProps }) {
  return (
    <>
      <CommandsField props={props} name="focused" label="Focused" />
      <CommandsField props={props} name="fast" label="Fast" />
      <CommandsField props={props} name="acceptance" label="Acceptance" />
      <TextField props={props} name="qa" label="QA command" fallback="none" />
      <TextField
        props={props}
        name="repair"
        label="Repair attempts"
        fallback={String(configDefaults.repair)}
        placeholder="1 to 8"
      />
      <p className="pt-2 text-xs text-muted-foreground">
        Test counts are read from what each check prints: vitest, jest, pytest,
        go test -v and go test -json. A check that prints no count shows its
        exit code only.
      </p>
    </>
  )
}
