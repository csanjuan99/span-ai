"use client"

import type { FormEvent, RefObject } from "react"

import { HowSection, type HowProps } from "@/components/start-run/HowSection"
import { StartAside, type AsideProps } from "@/components/start-run/StartAside"
import { WhatSection, type WhatProps } from "@/components/start-run/WhatSection"
import {
  WhereSection,
  type WhereProps,
} from "@/components/start-run/WhereSection"
import { twoColumns } from "@/components/shaping/session-ui"

/** Everything the start form shows and reports back, one block per step of the board. */
export interface StartFormProps {
  formRef: RefObject<HTMLFormElement | null>
  where: WhereProps
  what: WhatProps
  how: HowProps
  aside: AsideProps
  runNote: string
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}

/** N1: where, what and how, and what Start will do. */
export function StartForm({
  formRef,
  where,
  what,
  how,
  aside,
  runNote,
  onSubmit,
}: StartFormProps) {
  return (
    <form
      ref={formRef}
      className="flex flex-col gap-4"
      onSubmit={onSubmit}
      aria-label="Start a run"
    >
      <header className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight">Start a run</h1>
        <p className="text-[13px] text-muted-foreground">
          Choose the folder, the story and how to work. Nothing is created until
          you press Start.
        </p>
      </header>
      <div className={twoColumns}>
        <div className="flex min-w-0 flex-col gap-4">
          <WhereSection {...where} />
          <WhatSection what={what} />
          <HowSection {...how} />
          {runNote !== "" ? (
            <p role="alert" className="text-sm text-pretty text-destructive">
              {runNote}
            </p>
          ) : null}
        </div>
        <StartAside {...aside} />
      </div>
    </form>
  )
}
