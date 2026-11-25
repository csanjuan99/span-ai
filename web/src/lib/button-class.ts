/** The design's full-width button: 40px high with a 10px radius, 44px on a phone so a thumb lands on it. */
export const wideButton = "h-10 w-full rounded-[10px] max-sm:h-11"

/** The phone-first button of the Shape boards: 44px on a phone, 40px from a tablet up. */
export const touchButton = "h-11 md:h-10"

/** {@link touchButton} stretched to the width of its column. */
export const touchWideButton = "h-11 w-full md:h-10"

/** {@link touchButton} for a compact control: 44px on a phone, 28px from a tablet up. */
export const touchButtonXs = "h-11 md:h-7"

/** {@link touchButton} for a small control: 44px on a phone, 32px from a tablet up. */
export const touchButtonSm = "h-11 md:h-8"

/** The design's quiet button: the raised fill and the strong border, not the tinted outline of the kit. */
export const quietButton =
  "border-input bg-card text-foreground hover:bg-muted dark:border-input dark:bg-card dark:hover:bg-muted"

/** The design's danger button: the quiet button with the fail color on its border and text. */
export const dangerButton =
  "border-fail bg-card text-fail hover:bg-fail-soft hover:text-fail dark:border-fail dark:bg-card dark:hover:bg-fail-soft"

/** Grows the touch area of a small inline link or control to 44px on a phone without moving anything around it. */
export const touchArea =
  "relative after:absolute after:-inset-x-1 after:-inset-y-3 md:after:hidden"
