import * as React from "react"
import { Collapsible as CollapsiblePrimitive } from "radix-ui"

function CollapsibleContent({
  ...props
}: React.ComponentProps<typeof CollapsiblePrimitive.Content>) {
  return (
    <CollapsiblePrimitive.CollapsibleContent
      data-slot="collapsible-content"
      {...props}
    />
  )
}

export { CollapsibleContent }
