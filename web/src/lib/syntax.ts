/** What a piece of a code line is, so the page can colour it. */
export type TokenKind =
  "plain" | "keyword" | "string" | "number" | "function" | "comment"

/** One coloured piece of a line. */
export interface Token {
  kind: TokenKind
  text: string
}

const words: Record<string, readonly string[]> = {
  go: [
    "func",
    "return",
    "if",
    "else",
    "for",
    "range",
    "switch",
    "case",
    "default",
    "type",
    "struct",
    "interface",
    "package",
    "import",
    "var",
    "const",
    "go",
    "defer",
    "chan",
    "map",
    "nil",
    "true",
    "false",
    "err",
  ],
  script: [
    "function",
    "return",
    "if",
    "else",
    "for",
    "while",
    "switch",
    "case",
    "default",
    "const",
    "let",
    "var",
    "import",
    "export",
    "from",
    "class",
    "extends",
    "new",
    "async",
    "await",
    "interface",
    "type",
    "null",
    "undefined",
    "true",
    "false",
    "try",
    "catch",
    "throw",
  ],
  python: [
    "def",
    "return",
    "if",
    "elif",
    "else",
    "for",
    "while",
    "import",
    "from",
    "class",
    "with",
    "as",
    "try",
    "except",
    "raise",
    "None",
    "True",
    "False",
    "lambda",
    "yield",
    "async",
    "await",
  ],
  rust: [
    "fn",
    "let",
    "mut",
    "pub",
    "struct",
    "enum",
    "impl",
    "match",
    "use",
    "mod",
    "return",
    "if",
    "else",
    "for",
    "while",
    "loop",
    "true",
    "false",
    "self",
    "Self",
  ],
  sql: [
    "select",
    "from",
    "where",
    "insert",
    "into",
    "update",
    "delete",
    "create",
    "table",
    "alter",
    "drop",
    "index",
    "values",
    "and",
    "or",
    "not",
    "null",
    "primary",
    "key",
  ],
}

const byExtension: Record<string, keyof typeof words> = {
  go: "go",
  ts: "script",
  tsx: "script",
  js: "script",
  jsx: "script",
  mjs: "script",
  py: "python",
  rs: "rust",
  sql: "sql",
}

const commentStart: Record<string, string> = {
  go: "//",
  script: "//",
  python: "#",
  rust: "//",
  sql: "--",
}

/** The language family a path's extension names, or null for a file this page does not colour. */
export function languageOf(path: string): keyof typeof words | null {
  const at = path.lastIndexOf(".")
  return at < 0 ? null : (byExtension[path.slice(at + 1).toLowerCase()] ?? null)
}

const piece =
  /("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|`[^`]*`)|(\b\d[\d_.]*\b)|([A-Za-z_][A-Za-z0-9_]*)(?=\()|([A-Za-z_][A-Za-z0-9_]*)/g

/** Splits one line of code into coloured pieces for the language of its file. A line of a file with no language is one plain piece. */
export function tokenize(
  line: string,
  language: keyof typeof words | null
): Token[] {
  if (language === null || line === "") {
    return [{ kind: "plain", text: line }]
  }
  const marker = commentStart[language] ?? "//"
  const cut = commentAt(line, marker)
  const code = cut < 0 ? line : line.slice(0, cut)
  const tokens: Token[] = []
  const keywords = new Set(words[language])
  let last = 0
  for (const match of code.matchAll(piece)) {
    const at = match.index
    if (at > last) {
      tokens.push({ kind: "plain", text: code.slice(last, at) })
    }
    const [whole, str, num, call, word] = match
    if (str !== undefined) {
      tokens.push({ kind: "string", text: whole })
    } else if (num !== undefined) {
      tokens.push({ kind: "number", text: whole })
    } else if (call !== undefined) {
      tokens.push({
        kind: keywords.has(call) ? "keyword" : "function",
        text: whole,
      })
    } else if (word !== undefined) {
      tokens.push({
        kind: keywords.has(word) ? "keyword" : "plain",
        text: whole,
      })
    }
    last = at + whole.length
  }
  if (last < code.length) {
    tokens.push({ kind: "plain", text: code.slice(last) })
  }
  if (cut >= 0) {
    tokens.push({ kind: "comment", text: line.slice(cut) })
  }
  return tokens
}

/** Where a line comment starts, ignoring a marker inside a string; -1 when there is none. */
function commentAt(line: string, marker: string): number {
  let quote = ""
  for (let index = 0; index < line.length; index += 1) {
    const char = line.charAt(index)
    if (quote !== "") {
      if (char === "\\") {
        index += 1
      } else if (char === quote) {
        quote = ""
      }
    } else if (char === '"' || char === "'" || char === "`") {
      quote = char
    } else if (line.startsWith(marker, index)) {
      return index
    }
  }
  return -1
}
