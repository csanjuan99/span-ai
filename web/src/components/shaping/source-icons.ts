import {
  BookOpenIcon,
  CodeIcon,
  FileTextIcon,
  MessageSquareIcon,
} from "lucide-react"

/** The icon of each kind of source an item of the example map can come from. */
export const sourceIcons = {
  prompt: FileTextIcon,
  answer: MessageSquareIcon,
  code: CodeIcon,
  context: BookOpenIcon,
} as const
