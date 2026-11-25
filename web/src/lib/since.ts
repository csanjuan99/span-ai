import { hhmm } from "@/lib/clock"

const months = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "May",
  "Jun",
  "Jul",
  "Aug",
  "Sep",
  "Oct",
  "Nov",
  "Dec",
]

function sameDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  )
}

/**
 * "held 14:36 · 16 min ago": when a state began and how long it has lasted. Older than today the
 * clock time and the day carry it, since "40 h ago" is harder to place than "yesterday 22:05".
 * Empty when the time is unknown or ahead of `now`, so a bad stamp says nothing.
 */
export function sinceLine(label: string, stamp: string, now: Date): string {
  const then = new Date(stamp)
  if (
    stamp === "" ||
    Number.isNaN(then.getTime()) ||
    then.getTime() > now.getTime()
  ) {
    return ""
  }
  if (!sameDay(then, now)) {
    const yesterday = new Date(
      now.getFullYear(),
      now.getMonth(),
      now.getDate() - 1
    )
    const day = sameDay(then, yesterday)
      ? "yesterday"
      : `${then.getDate()} ${months[then.getMonth()]}`
    return `${label} ${day} ${hhmm(then)}`
  }
  const minutes = Math.floor((now.getTime() - then.getTime()) / 60_000)
  const hours = Math.floor(minutes / 60)
  const ago =
    minutes < 1
      ? "just now"
      : hours < 1
        ? `${minutes} min ago`
        : `${hours} h${minutes % 60 === 0 ? "" : ` ${minutes % 60} min`} ago`
  return `${label} ${hhmm(then)} · ${ago}`
}
