"use client"

import { useEffect, useState } from "react"

import {
  filterOf,
  isFiltered,
  type ListFilter,
} from "@/components/shape-entry/SessionsBoard"
import { listShapes } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import type { ShapeFilter, ShapeSession } from "@/lib/types"

/** The sessions of a folder: all of them (for counts) and the ones the filter keeps. */
export interface ShapeSessions {
  all: ShapeSession[]
  rows: ShapeSession[]
  /** False until the first answer, so a page does not flash its empty state. */
  loaded: boolean
  note: string
  reload: () => void
}

/**
 * Reads the sessions of a folder. The server applies the filter; a typed search waits a
 * moment so each key does not ask again. `reload` reads again after a change.
 */
export function useShapeSessions(
  workspace: string,
  filter: ListFilter
): ShapeSessions {
  const [all, setAll] = useState<ShapeSession[]>([])
  const [rows, setRows] = useState<ShapeSession[]>([])
  const [loaded, setLoaded] = useState(false)
  const [note, setNote] = useState("")
  const [tick, setTick] = useState(0)
  const filtered = isFiltered(filter)
  const sent = JSON.stringify(filterOf(filter))

  useEffect(() => {
    let live = true
    void listShapes(workspace)
      .then((list) => {
        if (live) {
          setAll(list)
          // The rows stay what they were until the filtered answer arrives.
          setRows((now) => (now.length === 0 ? list : now))
          setLoaded(true)
          setNote("")
        }
      })
      .catch((err: unknown) => {
        if (live) {
          setLoaded(true)
          setNote(reasonText(err, "the sessions could not be read"))
        }
      })
    return () => {
      live = false
    }
  }, [workspace, tick])

  useEffect(() => {
    if (!filtered) {
      return
    }
    let live = true
    const timer = window.setTimeout(() => {
      void listShapes(workspace, JSON.parse(sent) as ShapeFilter)
        .then((list) => {
          if (live) {
            setRows(list)
          }
        })
        .catch((err: unknown) => {
          if (live) {
            setNote(reasonText(err, "the sessions could not be read"))
          }
        })
    }, 250)
    return () => {
      live = false
      window.clearTimeout(timer)
    }
  }, [workspace, filtered, sent, tick])

  return {
    all,
    rows: filtered ? rows : all,
    loaded,
    note,
    reload: () => setTick((n) => n + 1),
  }
}
