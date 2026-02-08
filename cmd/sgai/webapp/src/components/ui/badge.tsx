import * as React from "react"
import { cn } from "@/lib/utils"

interface BadgeProps extends React.ComponentProps<"span"> {
  variant?: "default" | "secondary" | "destructive" | "outline"
}

function Badge({
  className,
  variant = "default",
  ...props
}: BadgeProps) {
  const variantClasses: Record<string, string> = {
    default: "border-transparent bg-primary text-primary-foreground shadow-sm",
    secondary: "border-transparent bg-secondary text-secondary-foreground",
    destructive: "border-transparent bg-destructive text-white shadow-sm",
    outline: "text-foreground",
  }

  return (
    <span
      data-slot="badge"
      className={cn(
        "inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs font-semibold transition-colors",
        variantClasses[variant],
        className
      )}
      {...props}
    />
  )
}

export { Badge }
export type { BadgeProps }
