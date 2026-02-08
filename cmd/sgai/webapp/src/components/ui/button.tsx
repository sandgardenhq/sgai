import * as React from "react"
import { cn } from "@/lib/utils"

interface ButtonProps extends React.ComponentProps<"button"> {
  variant?: "default" | "destructive" | "outline" | "secondary" | "ghost" | "link"
  size?: "default" | "sm" | "lg" | "icon"
  asChild?: boolean
}

function Button({
  className,
  variant = "default",
  size = "default",
  asChild = false,
  children,
  ...props
}: ButtonProps) {
  const variantClasses: Record<string, string> = {
    default: "bg-primary text-primary-foreground shadow-xs hover:bg-primary/90",
    destructive: "bg-destructive text-white shadow-xs hover:bg-destructive/90",
    outline: "border bg-background shadow-xs hover:bg-accent hover:text-accent-foreground",
    secondary: "bg-secondary text-secondary-foreground shadow-xs hover:bg-secondary/80",
    ghost: "hover:bg-accent hover:text-accent-foreground",
    link: "text-primary underline-offset-4 hover:underline",
  }

  const sizeClasses: Record<string, string> = {
    default: "h-9 px-4 py-2 text-sm",
    sm: "h-8 rounded-md px-3 text-xs",
    lg: "h-10 rounded-md px-6 text-sm",
    icon: "size-9",
  }

  const classes = cn(
    "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md font-medium transition-all disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0",
    variantClasses[variant],
    sizeClasses[size],
    className
  )

  if (asChild && React.isValidElement(children)) {
    return React.cloneElement(children, {
      ...props,
      "data-slot": "button",
      className: cn(classes, children.props.className),
    })
  }

  return (
    <button
      data-slot="button"
      className={classes}
      {...props}
    >
      {children}
    </button>
  )
}

export { Button }
export type { ButtonProps }
