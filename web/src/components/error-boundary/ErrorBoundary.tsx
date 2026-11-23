import { Component, type ErrorInfo, type ReactNode } from "react"

import { Button } from "@/components/ui/button"

/** What the boundary wraps, and the value whose change means "a different page". */
export interface ErrorBoundaryProps {
  children: ReactNode
  /** When it changes (for example the route), a caught fault is cleared. */
  resetKey?: string
  /** Reloads the whole page; injected by tests, `window.location.reload` otherwise. */
  reload?: () => void
}

interface ErrorBoundaryState {
  error: Error | null
}

function asError(value: unknown): Error {
  return value instanceof Error ? value : new Error(String(value))
}

/**
 * Catches a render fault so one broken component shows a message, Try again and Reload page
 * instead of blanking the whole desk. React only offers this as a class component, which
 * is why this is the one class in the app. The fault is still logged, never swallowed.
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  override state: ErrorBoundaryState = { error: null }

  static getDerivedStateFromError(error: unknown): ErrorBoundaryState {
    return { error: asError(error) }
  }

  override componentDidCatch(error: unknown, info: ErrorInfo): void {
    console.error("the page failed to render", asError(error), info.componentStack)
  }

  override componentDidUpdate(previous: ErrorBoundaryProps): void {
    if (this.state.error !== null && previous.resetKey !== this.props.resetKey) {
      this.reset()
    }
  }

  reset = (): void => {
    this.setState({ error: null })
  }

  // Try again only re-renders, so a fault that is not transient comes straight back;
  // reloading fetches the page and its data again.
  reloadPage = (): void => {
    const { reload } = this.props
    if (reload) {
      reload()
      return
    }
    window.location.reload()
  }

  override render(): ReactNode {
    const { error } = this.state
    if (error === null) {
      return this.props.children
    }
    return (
      <div role="alert" className="flex flex-col gap-3 p-6">
        <p className="text-pretty text-sm">Something on this page broke.</p>
        <pre className="whitespace-pre-wrap text-sm text-destructive">{error.message}</pre>
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant="secondary" onClick={this.reset}>
            Try again
          </Button>
          <Button type="button" variant="secondary" onClick={this.reloadPage}>
            Reload page
          </Button>
        </div>
      </div>
    )
  }
}
