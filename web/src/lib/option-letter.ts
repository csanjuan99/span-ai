/** The letter that names an option on a card: A, B, C… */
export function optionLetter(at: number): string {
  return String.fromCharCode(65 + at)
}
