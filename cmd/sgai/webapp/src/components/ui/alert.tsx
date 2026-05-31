import * as React from "react"

import { cn } from "@/lib/utils"
import { AlertDescription } from "./alert-description"
import { AlertTitle } from "./alert-title"

function Alert({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="alert"
      role="alert"
      className={cn(
        "relative w-full rounded-lg border px-4 py-3 text-sm [&>svg+div]:translate-y-[-3px] [&>svg]:absolute [&>svg]:left-4 [&>svg]:top-4 [&>svg]:text-foreground [&>svg~*]:pl-7",
        className
      )}
      {...props}
    />
  )
}

export { Alert, AlertTitle, AlertDescription }
