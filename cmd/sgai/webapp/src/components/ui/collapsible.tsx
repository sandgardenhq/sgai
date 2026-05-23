import * as React from "react"
import { Collapsible as CollapsiblePrimitive } from "radix-ui"
import { CollapsibleTrigger } from "./collapsible-trigger"
import { CollapsibleContent } from "./collapsible-content"

function Collapsible({
  ...props
}: React.ComponentProps<typeof CollapsiblePrimitive.Root>) {
  return <CollapsiblePrimitive.Root data-slot="collapsible" {...props} />
}

export { Collapsible, CollapsibleTrigger, CollapsibleContent }
