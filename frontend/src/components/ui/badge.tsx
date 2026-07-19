import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { Slot } from "radix-ui"

import { cn } from "@/lib/utils"

// Extends shadcn defaults with the palette tokens from
// DESIGN_SPEC_REFINED.md ("Achievement Gold", "Badges": bronze/silver/gold/diamond).
const badgeVariants = cva(
  "inline-flex w-fit shrink-0 items-center justify-center gap-1 overflow-hidden rounded-full border border-transparent px-2 py-0.5 text-xs font-medium whitespace-nowrap transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 aria-invalid:border-destructive aria-invalid:ring-destructive/20 [&>svg]:pointer-events-none [&>svg]:size-3",
  {
    variants: {
      variant: {
        default: "bg-primary text-white [a&]:hover:bg-primary-hover",
        secondary: "bg-surface-elevated text-text-secondary",
        success: "bg-success-tint text-success",
        danger: "bg-danger-tint text-danger",
        warning: "bg-warning/12 text-warning",
        info: "bg-info/12 text-info",
        gold: "bg-gold-glow text-gold",
        silver: "bg-silver/15 text-silver",
        bronze: "bg-bronze/15 text-bronze",
        diamond: "bg-diamond/15 text-diamond",
        outline:
          "border-border text-text [a&]:hover:bg-surface-hover",
        ghost: "[a&]:hover:bg-surface-hover",
        link: "text-primary underline-offset-4 [a&]:hover:underline",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)

function Badge({
  className,
  variant = "default",
  asChild = false,
  ...props
}: React.ComponentProps<"span"> &
  VariantProps<typeof badgeVariants> & { asChild?: boolean }) {
  const Comp = asChild ? Slot.Root : "span"

  return (
    <Comp
      data-slot="badge"
      data-variant={variant}
      className={cn(badgeVariants({ variant }), className)}
      {...props}
    />
  )
}

export { Badge, badgeVariants }
