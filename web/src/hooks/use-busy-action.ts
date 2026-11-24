import { useState } from "react"

/**
 * Runs one slow action at a time. `busy` is true while it runs, `onStart` clears whatever the
 * last attempt left on screen, and `onError` receives what the action threw with the fallback
 * text the caller gave for it.
 */
export function useBusyAction({
  onStart,
  onError,
  keepBusyOnSuccess = false,
}: {
  onStart?: () => void
  onError: (err: unknown, fallback: string) => void
  /** For an action that leaves the page when it works: busy stays on so nothing can resubmit. */
  keepBusyOnSuccess?: boolean
}) {
  const [busy, setBusy] = useState(false)

  async function run(
    task: () => Promise<void>,
    fallback = "the request failed"
  ) {
    setBusy(true)
    onStart?.()
    try {
      await task()
      if (!keepBusyOnSuccess) {
        setBusy(false)
      }
    } catch (err: unknown) {
      onError(err, fallback)
      setBusy(false)
    }
  }

  return { busy, run }
}
