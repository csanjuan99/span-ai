/** How much the page moves: follow the system, or never play the start-up and motion at all. */
export type MotionPref = "system" | "reduced"

const key = "span:motion"
const styleId = "span-motion"

/** What the person chose, or "system" when nothing was saved or storage is blocked. */
export function readMotion(): MotionPref {
  try {
    return window.localStorage.getItem(key) === "reduced" ? "reduced" : "system"
  } catch {
    return "system"
  }
}

/** Remembers the choice and applies it. False when storage refused, in which case it still applies for this load. */
export function saveMotion(pref: MotionPref): boolean {
  let stored = true
  try {
    window.localStorage.setItem(key, pref)
  } catch {
    stored = false
  }
  applyMotion(pref)
  return stored
}

/** Whether the page should hold still: the person asked, or the system did. */
export function motionReduced(): boolean {
  return (
    readMotion() === "reduced" ||
    window.matchMedia("(prefers-reduced-motion: reduce)").matches
  )
}

/**
 * Puts the choice on the page. "reduced" adds one rule that ends every animation and
 * transition at once, so shimmer, spinners and fades stop without each component asking.
 */
export function applyMotion(pref: MotionPref = readMotion()): void {
  const present = document.getElementById(styleId)
  if (pref !== "reduced") {
    present?.remove()
    return
  }
  if (present !== null) {
    return
  }
  const style = document.createElement("style")
  style.id = styleId
  style.textContent =
    "*,*::before,*::after{animation-duration:.01ms!important;animation-iteration-count:1!important;transition-duration:.01ms!important}"
  document.head.appendChild(style)
}
