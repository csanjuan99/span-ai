import { groupOf, pillOf } from "@/lib/desk"
import { isRecord } from "@/lib/guards"
import type { SpanHead } from "@/lib/types"

/** What a person is told about: a run needs them, a check failed, or a run finished. */
export type NotifyKind = "needs" | "failed" | "finished" | "blocked" | "stopped"

/** One thing that just happened to one run. */
export interface NotifyEvent {
  kind: NotifyKind
  run: SpanHead
}

/** How one kind of event reaches a person. `system` is the browser's own notification. */
export interface Channels {
  toast: boolean
  tab: boolean
  system: boolean
}

export interface NotifyPrefs {
  events: Record<NotifyKind, Channels>
  /** Hold toasts back while this tab is in front. */
  background: boolean
  /** A soft tone with each toast. */
  sound: boolean
}

/** The kinds in the order a person reads them. */
export const notifyKinds: NotifyKind[] = [
  "needs",
  "failed",
  "finished",
  "blocked",
  "stopped",
]

/** What each kind is called on screen, with the words that explain it. */
export const kindCopy: Record<NotifyKind, { label: string; hint: string }> = {
  needs: {
    label: "A run needs me",
    hint: "question, decision, approval or conflict",
  },
  failed: {
    label: "A check failed",
    hint: "verify or qa ended with a failing exit",
  },
  finished: { label: "A run finished", hint: "merged and sealed" },
  blocked: { label: "Agent blocked", hint: "login expired or quota used up" },
  stopped: { label: "A run stopped", hint: "by you or by Span" },
}

/**
 * What a run is, as a person would name it. The engine words a stop and an agent block in its
 * reason, so those two are read from the start of it: "agent credentials missing", "agent quota
 * exhausted", "the run stopped", "stopped by a decision". Anything else keeps its group's kind.
 */
export function kindOf(run: SpanHead): NotifyKind | undefined {
  const group = groupOf(run)
  const reason = run.reason.trim().toLowerCase()
  if (group === "waiting") {
    return /^agent (credentials missing|quota exhausted)/.test(reason)
      ? "blocked"
      : "needs"
  }
  if (group === "failed") {
    return /^(the run stopped|stopped by)/.test(reason) ? "stopped" : "failed"
  }
  return group === "done" ? "finished" : undefined
}

/**
 * The runs that moved into waiting, failed, stopped, blocked or done since the last read. The
 * first read has nothing to compare with, so it announces nothing: opening the desk is not news.
 */
export function notifyEvents(
  previous: SpanHead[] | null,
  next: SpanHead[]
): NotifyEvent[] {
  if (previous === null) {
    return []
  }
  const before = new Map(previous.map((run) => [run.id, kindOf(run)]))
  const events: NotifyEvent[] = []
  for (const run of next) {
    const kind = kindOf(run)
    if (kind !== undefined && before.get(run.id) !== kind) {
      events.push({ kind, run })
    }
  }
  return events.sort(
    (a, b) => notifyKinds.indexOf(a.kind) - notifyKinds.indexOf(b.kind)
  )
}

/** How many runs need a person: the ones waiting and the ones that failed. */
export function attentionCount(runs: SpanHead[]): number {
  return runs.filter(
    (run) => groupOf(run) === "waiting" || groupOf(run) === "failed"
  ).length
}

function nameOf(run: SpanHead): string {
  return run.issue !== "" ? run.issue : run.title !== "" ? run.title : run.id
}

/**
 * The mark on the favicon: how many runs wait for a person, `!` when only failures do, `✓` for
 * a run that just finished, and nothing when all is quiet. Never colour alone.
 */
export function badgeOf(runs: SpanHead[], finished: SpanHead | null): string {
  const waiting = runs.filter((run) => groupOf(run) === "waiting").length
  if (waiting > 0) {
    return String(waiting)
  }
  if (runs.some((run) => groupOf(run) === "failed")) {
    return "!"
  }
  return finished !== null ? "✓" : ""
}

/** The tab title: the product name, led by the mark and the first run that needs a person. */
export function titleOf(runs: SpanHead[], finished: SpanHead | null): string {
  const mark = badgeOf(runs, finished)
  if (mark === "" || (finished === null && attentionCount(runs) === 0)) {
    return "Span"
  }
  const waiting = runs.find((run) => groupOf(run) === "waiting")
  if (waiting !== undefined) {
    return `(${mark}) ${nameOf(waiting)} needs you · Span`
  }
  const failed = runs.find((run) => groupOf(run) === "failed")
  if (failed !== undefined) {
    return `(${mark}) ${nameOf(failed)} failed · Span`
  }
  return finished === null
    ? "Span"
    : `(${mark}) ${nameOf(finished)} done · Span`
}

const storageKey = "span:notify"

/** Toasts and the tab on, the browser's own notification off until asked, toasts only when away. */
export const defaultPrefs: NotifyPrefs = {
  events: {
    needs: { toast: true, tab: true, system: false },
    failed: { toast: true, tab: true, system: false },
    finished: { toast: true, tab: true, system: false },
    blocked: { toast: true, tab: true, system: false },
    stopped: { toast: false, tab: true, system: false },
  },
  background: true,
  sound: false,
}

function channelsOf(value: unknown, fallback: Channels): Channels {
  const row = isRecord(value) ? value : {}
  return {
    toast: typeof row.toast === "boolean" ? row.toast : fallback.toast,
    tab: typeof row.tab === "boolean" ? row.tab : fallback.tab,
    system: typeof row.system === "boolean" ? row.system : fallback.system,
  }
}

/** The saved choices. A missing, blocked or damaged value gives the defaults for what it lacks. */
export function loadPrefs(): NotifyPrefs {
  let raw: string | null
  try {
    raw = window.localStorage.getItem(storageKey)
  } catch {
    return defaultPrefs
  }
  if (raw === null) {
    return defaultPrefs
  }
  let body: unknown
  try {
    body = JSON.parse(raw)
  } catch {
    return defaultPrefs
  }
  const saved = isRecord(body) ? body : {}
  const events = isRecord(saved.events) ? saved.events : {}
  return {
    events: {
      needs: channelsOf(events.needs, defaultPrefs.events.needs),
      failed: channelsOf(events.failed, defaultPrefs.events.failed),
      finished: channelsOf(events.finished, defaultPrefs.events.finished),
      blocked: channelsOf(events.blocked, defaultPrefs.events.blocked),
      stopped: channelsOf(events.stopped, defaultPrefs.events.stopped),
    },
    background:
      typeof saved.background === "boolean"
        ? saved.background
        : defaultPrefs.background,
    sound: typeof saved.sound === "boolean" ? saved.sound : defaultPrefs.sound,
  }
}

/** The event the page fires after a save, so the hook and the settings agree without a reload. */
export const prefsChanged = "span:notify-prefs"

/** Saves the choices. False when the browser refused to keep them. */
export function savePrefs(prefs: NotifyPrefs): boolean {
  try {
    window.localStorage.setItem(storageKey, JSON.stringify(prefs))
  } catch {
    return false
  }
  window.dispatchEvent(new Event(prefsChanged))
  return true
}

/** Whether a toast should show now: this kind has toasts on, and the tab is not in front when asked. */
export function wantsToast(
  prefs: NotifyPrefs,
  kind: NotifyKind,
  tabInFront: boolean
): boolean {
  return prefs.events[kind].toast && !(prefs.background && tabInFront)
}

/** What a toast says: who, what happened, and the word on its button. */
export function toastCopy(event: NotifyEvent): {
  title: string
  detail: string
  action: string
} {
  const name = nameOf(event.run)
  if (event.kind === "needs") {
    return {
      title: `${name} needs you`,
      detail: pillOf(event.run),
      action: "Review",
    }
  }
  if (event.kind === "failed") {
    return {
      title: `${name} check failed`,
      detail: pillOf(event.run),
      action: "Open",
    }
  }
  if (event.kind === "blocked") {
    return {
      title: `${name} agent blocked`,
      detail: pillOf(event.run),
      action: "Open",
    }
  }
  if (event.kind === "stopped") {
    return {
      title: `${name} stopped`,
      detail: pillOf(event.run),
      action: "Open",
    }
  }
  return {
    title: `${name} is done`,
    detail: event.run.title !== "" ? event.run.title : "merged and sealed",
    action: "Summary",
  }
}

const iconSize = 64

/**
 * The page icon with a mark in its corner, as a data URL. Empty when the browser cannot draw
 * (no canvas), in which case the original icon stays.
 */
export function faviconWith(mark: string): string {
  const canvas = document.createElement("canvas")
  canvas.width = iconSize
  canvas.height = iconSize
  const ctx = canvas.getContext("2d")
  if (ctx === null) {
    return ""
  }
  ctx.fillStyle = "#5b3fd4"
  ctx.beginPath()
  ctx.roundRect(4, 4, iconSize - 8, iconSize - 8, 14)
  ctx.fill()
  ctx.fillStyle = "#ffffff"
  ctx.font = "bold 34px sans-serif"
  ctx.textAlign = "center"
  ctx.textBaseline = "middle"
  ctx.fillText("S", 28, 34)
  if (mark !== "") {
    ctx.fillStyle = "#c2243d"
    ctx.beginPath()
    ctx.arc(46, 46, 17, 0, Math.PI * 2)
    ctx.fill()
    ctx.fillStyle = "#ffffff"
    ctx.font = "bold 22px sans-serif"
    ctx.fillText(mark, 46, 47)
  }
  return canvas.toDataURL("image/png")
}

/**
 * A soft two-note tone, drawn with the browser's own audio so nothing is downloaded. It does
 * nothing when the browser has no audio or refuses to start it before a click.
 */
export function playTone(): void {
  const Context =
    window.AudioContext ??
    (window as unknown as { webkitAudioContext?: typeof AudioContext })
      .webkitAudioContext
  if (Context === undefined) {
    return
  }
  try {
    const ctx = new Context()
    const gain = ctx.createGain()
    gain.gain.setValueAtTime(0.0001, ctx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.08, ctx.currentTime + 0.02)
    gain.gain.exponentialRampToValueAtTime(0.0001, ctx.currentTime + 0.5)
    gain.connect(ctx.destination)
    for (const [index, hz] of [660, 880].entries()) {
      const osc = ctx.createOscillator()
      osc.type = "sine"
      osc.frequency.value = hz
      osc.connect(gain)
      osc.start(ctx.currentTime + index * 0.16)
      osc.stop(ctx.currentTime + index * 0.16 + 0.2)
    }
    window.setTimeout(() => void ctx.close(), 900)
  } catch {
    // Audio is a courtesy: a refusal changes nothing a person needs.
  }
}
