/** Reads an RFC 3339 instant. Null when it is empty or not a date, so a bad stamp says nothing. */
export function instantOf(iso: string | undefined): Date | null {
  if (iso === undefined || iso === "") {
    return null
  }
  const at = new Date(iso)
  return Number.isNaN(at.getTime()) ? null : at
}

function pair(value: number): string {
  return value.toString().padStart(2, "0")
}

/** "09:36" in the viewer's zone. */
export function hhmm(date: Date): string {
  return `${pair(date.getHours())}:${pair(date.getMinutes())}`
}

/** "09:36:05" in the viewer's zone. */
export function hhmmss(date: Date): string {
  return `${hhmm(date)}:${pair(date.getSeconds())}`
}

/** The clock time of an instant as "09:36", or nothing when the stamp is not a date. */
export function clockTime(iso: string | undefined): string {
  const at = instantOf(iso)
  return at === null ? "" : hhmm(at)
}

/** The clock time of an instant with seconds, "09:36:05", or nothing when it is not a date. */
export function clockSeconds(iso: string | undefined): string {
  const at = instantOf(iso)
  return at === null ? "" : hhmmss(at)
}

/** The day of an instant as "9 Jun", or nothing when the stamp is not a date. */
export function dayLabel(iso: string | undefined): string {
  const at = instantOf(iso)
  return at === null
    ? ""
    : at.toLocaleDateString("en-GB", { day: "numeric", month: "short" })
}

/** The time as the person reads it: the hour today, the day and hour before. */
export function whenLabel(iso: string | undefined, now = new Date()): string {
  const at = instantOf(iso)
  if (at === null) {
    return ""
  }
  const hour = hhmm(at)
  return at.toDateString() === now.toDateString()
    ? hour
    : `${dayLabel(iso)} ${hour}`
}
