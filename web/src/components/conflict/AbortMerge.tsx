import { AlertDialog } from "radix-ui"

import { Button } from "@/components/ui/button"

/**
 * Abort behind a confirmation: `git merge --abort` throws away every choice made so far,
 * so one stray click must not run it.
 */
export function AbortMerge({ onConfirm }: { onConfirm: () => void }) {
  return (
    <AlertDialog.Root>
      <AlertDialog.Trigger asChild>
        <Button type="button" variant="destructive" className="self-start">
          Abort merge
        </Button>
      </AlertDialog.Trigger>
      <AlertDialog.Portal>
        <AlertDialog.Overlay className="fixed inset-0 bg-background/80" />
        <AlertDialog.Content className="fixed top-1/2 left-1/2 flex w-[min(28rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 flex-col gap-3 border border-border bg-card p-4">
          <AlertDialog.Title className="text-balance">Abort the merge?</AlertDialog.Title>
          <AlertDialog.Description className="text-pretty text-sm text-muted-foreground">
            This runs git merge --abort. The branch is left exactly as it was, and the run holds until you resume it.
          </AlertDialog.Description>
          <div className="flex flex-wrap justify-end gap-2">
            <AlertDialog.Cancel asChild>
              <Button type="button" variant="secondary">
                Keep resolving
              </Button>
            </AlertDialog.Cancel>
            <AlertDialog.Action asChild>
              <Button type="button" variant="destructive" onClick={onConfirm}>
                Abort the merge
              </Button>
            </AlertDialog.Action>
          </div>
        </AlertDialog.Content>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  )
}
