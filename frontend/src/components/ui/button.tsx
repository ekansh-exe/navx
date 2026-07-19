import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { Slot } from "radix-ui"
import { Loader2 } from "lucide-react"

import { cn } from "@/lib/utils"

// Variants follow DESIGN_SPEC_REFINED.md section 4 ("Buttons"):
// primary = filled blue, buy = filled green, sell = outlined red
// (never filled, to reduce accidental selling), secondary = gray outline,
// destructive = filled red for destructive actions only.
const buttonVariants = cva(
  "inline-flex shrink-0 items-center justify-center gap-2 rounded-button text-sm font-medium whitespace-nowrap transition-all outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:pointer-events-none disabled:opacity-40 aria-invalid:border-destructive aria-invalid:ring-destructive/20 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default:
          "bg-primary text-white hover:bg-primary-hover active:bg-primary-pressed",
        buy: "bg-success text-white hover:bg-success-hover active:brightness-90",
        sell: "border border-danger text-danger bg-transparent hover:bg-danger-tint active:brightness-90",
        secondary:
          "border border-border text-text bg-transparent hover:bg-surface-hover",
        destructive: "bg-danger text-white hover:bg-danger-hover",
        ghost: "hover:bg-surface-hover hover:text-text",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-12 px-5 has-[>svg]:px-4",
        sm: "h-9 gap-1.5 rounded-button px-3 has-[>svg]:px-2.5",
        xs: "h-6 gap-1 rounded-button px-2 text-xs has-[>svg]:px-1.5 [&_svg:not([class*='size-'])]:size-3",
        lg: "h-12 px-6 has-[>svg]:px-5",
        icon: "size-12",
        "icon-sm": "size-9",
        "icon-xs": "size-6 rounded-button [&_svg:not([class*='size-'])]:size-3",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

function Button({
  className,
  variant = "default",
  size = "default",
  asChild = false,
  loading = false,
  loadingText = "Executing...",
  disabled,
  children,
  ...props
}: React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean
    loading?: boolean
    loadingText?: string
  }) {
  const Comp = asChild ? Slot.Root : "button"

  return (
    <Comp
      data-slot="button"
      data-variant={variant}
      data-size={size}
      className={cn(
        buttonVariants({ variant, size, className }),
        loading && "opacity-70 pointer-events-none"
      )}
      disabled={disabled || loading}
      {...props}
    >
      {loading ? (
        <>
          <Loader2 className="animate-spin" />
          {loadingText}
        </>
      ) : (
        children
      )}
    </Comp>
  )
}

export { Button, buttonVariants }
