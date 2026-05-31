import * as React from "react"
import { cn } from "@/lib/utils"

function AlertTitle({ className, children, ...props }: React.ComponentProps<"h5">) {
  return (
    <h5
      data-slot="alert-title"
      className={cn("mb-1 font-medium leading-none tracking-tight", className)}
      {...props}
    >
      {children}
    </h5>
  )
}

export { AlertTitle }
