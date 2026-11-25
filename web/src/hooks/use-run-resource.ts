import { useEffect, useState } from "react"

import { reasonText } from "@/lib/guards"

/** A read of something about a run: on its way, here, or refused with a reason. */
export type Resource<T> =
  | { status: "loading" }
  | { status: "ready"; value: T }
  | { status: "error"; message: string }

/** What a resource hook hands back: the state, a way to read again, and a way to set the value after a write. */
export interface RunResource<T> {
  state: Resource<T>
  reload: () => void
  set: (value: T) => void
}

/**
 * Reads one thing about a run (its files, commits, reasoning or audit) when the run or the
 * reader changes, and again on reload. The reader must be a module-level function so its
 * identity is stable.
 */
export function useRunResource<T>(
  id: string,
  read: (id: string) => Promise<T>
): RunResource<T> {
  const [state, setState] = useState<Resource<T>>({ status: "loading" })
  const [tick, setTick] = useState(0)
  useEffect(() => {
    let closed = false
    void read(id)
      .then((value) => {
        if (!closed) {
          setState({ status: "ready", value })
        }
      })
      .catch((err: unknown) => {
        if (!closed) {
          setState({
            status: "error",
            message: reasonText(err, "could not read this"),
          })
        }
      })
    return () => {
      closed = true
    }
  }, [id, read, tick])
  return {
    state,
    reload: () => setTick((now) => now + 1),
    set: (value) => setState({ status: "ready", value }),
  }
}
