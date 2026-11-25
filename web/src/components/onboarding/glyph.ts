/** The mark of a step or stage: done, the one in progress, or still to come. */
export function stepGlyph(done: boolean, current: boolean): string {
  return done ? "+" : current ? ">" : "·"
}
