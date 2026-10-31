"use client"

import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { Tabs as TabsPrimitive } from "radix-ui"

import { cn } from "@/lib/utils"

function Tabs({
  className,
  orientation = "horizontal",
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Root>) {
  return (
    <TabsPrimitive.Root
      data-slot="tabs"
      data-orientation={orientation}
      className={cn(
        "group/tabs flex gap-(--gap-inner) data-horizontal:flex-col",
        className
      )}
      {...props}
    />
  )
}

const tabsListVariants = cva(
  "group/tabs-list relative inline-flex w-fit items-center justify-start gap-1 rounded-full p-1 [scrollbar-width:none] text-muted-foreground group-data-horizontal/tabs:h-(--control-h) group-data-vertical/tabs:h-fit group-data-vertical/tabs:flex-col data-[variant=line]:h-auto data-[variant=line]:rounded-none data-[variant=line]:p-0",
  {
    variants: {
      variant: {
        default: "bg-muted/80 shadow-[inset_0_1px_2px_rgba(107,42,110,0.08)]",
        line: "gap-1 bg-transparent",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)

/** Where the sliding pill sits: over the active trigger, measured in the list. */
function useActivePill(enabled: boolean) {
  const ref = React.useRef<HTMLDivElement>(null)
  const [pill, setPill] = React.useState<React.CSSProperties | null>(null)

  React.useLayoutEffect(() => {
    const list = ref.current
    if (!enabled || !list) return
    const place = () => {
      const active = list.querySelector<HTMLElement>(
        '[data-slot="tabs-trigger"][data-state="active"]'
      )
      setPill(
        active
          ? {
              width: active.offsetWidth,
              height: active.offsetHeight,
              transform: `translate(${active.offsetLeft}px, ${active.offsetTop}px)`,
            }
          : null
      )
    }
    place()
    const mutations = new MutationObserver(place)
    mutations.observe(list, { attributes: true, attributeFilter: ["data-state"], childList: true, subtree: true })
    const resizes = new ResizeObserver(place)
    resizes.observe(list)
    return () => {
      mutations.disconnect()
      resizes.disconnect()
    }
  }, [enabled])

  return { ref, pill }
}

function TabsList({
  className,
  variant = "default",
  children,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.List> &
  VariantProps<typeof tabsListVariants>) {
  const { ref, pill } = useActivePill(variant === "default")
  return (
    <TabsPrimitive.List
      ref={ref}
      data-slot="tabs-list"
      data-variant={variant}
      data-sliding={pill ? "" : undefined}
      className={cn(tabsListVariants({ variant }), className)}
      {...props}
    >
      {pill ? (
        <span
          aria-hidden
          data-slot="tabs-pill"
          style={pill}
          className="pointer-events-none absolute top-0 left-0 rounded-full bg-card shadow-[0_2px_8px_-2px_rgba(107,42,110,0.18)] transition-[transform,width,height] duration-300 ease-[cubic-bezier(0.22,1,0.36,1)] dark:bg-secondary"
        />
      ) : null}
      {children}
    </TabsPrimitive.List>
  )
}

function TabsTrigger({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Trigger>) {
  return (
    <TabsPrimitive.Trigger
      data-slot="tabs-trigger"
      className={cn(
        "relative inline-flex h-full flex-1 items-center justify-center gap-1.5 rounded-full border border-transparent px-4 text-[13px] font-medium whitespace-nowrap text-muted-foreground transition-all group-data-vertical/tabs:w-full group-data-vertical/tabs:justify-start hover:text-foreground focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-1 focus-visible:outline-ring disabled:pointer-events-none disabled:opacity-50 dark:text-muted-foreground dark:hover:text-foreground group-data-[variant=default]/tabs-list:data-active:shadow-sm group-data-[variant=line]/tabs-list:data-active:shadow-none [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        "group-data-[variant=line]/tabs-list:h-9 group-data-[variant=line]/tabs-list:rounded-none group-data-[variant=line]/tabs-list:px-3 group-data-[variant=line]/tabs-list:bg-transparent group-data-[variant=line]/tabs-list:data-active:bg-transparent dark:group-data-[variant=line]/tabs-list:data-active:border-transparent dark:group-data-[variant=line]/tabs-list:data-active:bg-transparent",
        "data-active:bg-card data-active:text-foreground data-active:shadow-[0_2px_8px_-2px_rgba(107,42,110,0.18)] dark:data-active:border-transparent dark:data-active:bg-secondary dark:data-active:text-foreground",
        "group-data-sliding/tabs-list:data-active:bg-transparent! group-data-sliding/tabs-list:data-active:shadow-none!",
        "after:absolute after:bg-foreground after:opacity-0 after:transition-opacity group-data-horizontal/tabs:after:inset-x-0 group-data-horizontal/tabs:after:bottom-[-5px] group-data-horizontal/tabs:after:h-0.5 group-data-vertical/tabs:after:inset-y-0 group-data-vertical/tabs:after:-right-1 group-data-vertical/tabs:after:w-0.5 group-data-[variant=line]/tabs-list:data-active:after:opacity-100",
        className
      )}
      {...props}
    />
  )
}

function TabsContent({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Content>) {
  return (
    <TabsPrimitive.Content
      data-slot="tabs-content"
      className={cn("flex-1 text-sm outline-none", className)}
      {...props}
    />
  )
}

export { Tabs, TabsList, TabsTrigger, TabsContent, tabsListVariants }
