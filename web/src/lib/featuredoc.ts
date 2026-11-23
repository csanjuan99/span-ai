/** A choice the agent says it made that the brief did not decide. */
export interface Decision {
  what: string
  /** The reason after "because". Empty when the agent gave none. */
  why: string
}

/** A way the agent says the change could hurt. */
export interface ReportedRisk {
  text: string
  /** True for "Risk: none found", which says the agent looked and found nothing. */
  none: boolean
}

/** One line of the Tasks section. */
export interface TaskItem {
  text: string
  done: boolean
}

/** What the ODD feature document says, read for the report. */
export interface FeatureReport {
  title: string
  specs: string[]
  tasks: TaskItem[]
  decisions: Decision[]
  risks: ReportedRisk[]
  /** True when the document has a Log section, whether or not it had Decision or Risk lines. */
  hasLog: boolean
}

const bullet = /^\s*(?:[-*]\s+|\d+[.)]\s+)?/

function sectionOf(line: string): string | null {
  const match = /^##\s+(.+?)\s*$/.exec(line)
  return match === null ? null : (match[1] ?? "").toLowerCase()
}

function stripMarks(text: string): string {
  return text.replace(/\*\*|__/g, "").trim()
}

/** Splits "<what> because <why>" on the first "because". */
export function splitDecision(text: string): Decision {
  const at = text.search(/\s(?:because|porque)\s/i)
  if (at < 0) {
    return { what: text.trim(), why: "" }
  }
  const rest = text.slice(at).trim().replace(/^(?:because|porque)\s+/i, "")
  return { what: text.slice(0, at).trim(), why: rest }
}

/** Reads the document the build stage leaves behind. Unknown shapes give empty lists, never an error. */
export function parseFeature(markdown: string): FeatureReport {
  const out: FeatureReport = { title: "", specs: [], tasks: [], decisions: [], risks: [], hasLog: false }
  let section = ""
  for (const raw of markdown.split("\n")) {
    const heading = sectionOf(raw)
    if (heading !== null) {
      section = heading
      if (heading === "log") {
        out.hasLog = true
      }
      continue
    }
    const title = /^#\s+(.+?)\s*$/.exec(raw)
    if (title !== null && out.title === "") {
      out.title = stripMarks(title[1] ?? "")
      continue
    }
    const line = raw.replace(bullet, "").trim()
    if (line === "") {
      continue
    }
    const tagged = /^(decision|risk)\s*:\s*(.+)$/i.exec(stripMarks(line))
    if (tagged !== null) {
      const body = (tagged[2] ?? "").trim()
      if ((tagged[1] ?? "").toLowerCase() === "decision") {
        out.decisions.push(splitDecision(body))
      } else {
        out.risks.push({ text: body, none: /^none( found)?\.?$/i.test(body) })
      }
      continue
    }
    if (section === "specs") {
      out.specs.push(stripMarks(line))
    } else if (section === "tasks") {
      const box = /^\[( |x|X)\]\s*(.*)$/.exec(line)
      out.tasks.push({ text: stripMarks(box === null ? line : (box[2] ?? "")), done: box !== null && box[1] !== " " })
    }
  }
  return out
}
