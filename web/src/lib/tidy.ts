// A color code, with the escape character or, when a log lost it on the way, without.
const escape = String.fromCharCode(27)
const color = new RegExp(`${escape}?\\[(?:\\d{1,3}(?:;\\d{1,3})*)?m`, "g")
// A Go duration as time.Duration prints it: 54m22.887124s, 1h3m5.5s, 12.4s.
const goDuration = /\b(?:(\d+)h)?(?:(\d+)m)?(\d+(?:\.\d+)?)s\b/g

/**
 * Makes text the engine wrote readable for a person: no terminal colors, and durations in whole
 * seconds (an hour or more drops the seconds). It never changes the meaning of a line.
 */
export function tidy(text: string): string {
  return text.replace(color, "").replace(goDuration, (match, hours?: string, minutes?: string, seconds?: string) => {
    if (hours === undefined && minutes === undefined && Number(seconds) < 1) {
      return match
    }
    const whole = Math.floor(Number(seconds))
    if (hours !== undefined) {
      return `${hours}h ${(minutes ?? "0").padStart(2, "0")}m`
    }
    return minutes === undefined ? `${whole}s` : `${minutes}m ${String(whole).padStart(2, "0")}s`
  })
}
