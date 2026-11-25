"use client"

import { useState } from "react"

import { useBusyAction } from "@/hooks/use-busy-action"
import { reasonText } from "@/lib/guards"

/** A busy action whose failure reads as a note on the page, cleared when the next one starts. */
export function useNotedAction() {
  const [note, setNote] = useState("")
  const { busy, run } = useBusyAction({
    onStart: () => setNote(""),
    onError: (err, fallback) => setNote(reasonText(err, fallback)),
  })
  return { note, setNote, busy, run }
}
