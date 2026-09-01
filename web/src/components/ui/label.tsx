import * as React from "react"

import { cn } from "@/lib/utils"

// 不挂 base-ui Field.Label:Field.Label 需要 Field.Root 上下文做自动 id 关联,但 Input/Textarea 是独立原语没 Field.Root,强用会触发 "FieldRootContext is missing" 报错。
function Label({ className, ...props }: React.ComponentProps<"label">) {
  return (
    <label
      data-slot="label"
      className={cn(
        "text-sm font-medium text-foreground select-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70",
        className
      )}
      {...props}
    />
  )
}

export { Label }