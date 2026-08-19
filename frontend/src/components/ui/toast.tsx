"use client";

import { Toast as ToastPrimitive } from "@base-ui/react/toast";
import { CheckCircle2, AlertCircle, XCircle, Info, X } from "lucide-react";
import { cn } from "@/lib/utils";

// ToastItem renders a single toast notification. Uses base-ui's
// primitives (Root, Title, Description, Close) so the close button,
// focus management and ARIA live-region behaviour are handled by
// the library. Visual styling (icon, color, border) is our own and
// is driven by the toast's `type` field.
//
// Type contract:
//   - "success": green (emerald). Auto-closes at 5s.
//   - "warning": amber. Stays until the user clicks X.
//   - "error":   red (destructive). Stays until the user clicks X.
//   - "info":    sky. Auto-closes at 5s.
//
// The timeout is set by the toast manager when the toast is added;
// this component does not duplicate that policy.
//
// `toast` is REQUIRED by base-ui's Root primitive — it carries the
// id and metadata that the manager needs to track the toast's
// lifecycle (auto-close, focus restoration, swipe-to-dismiss).
// Without it, Root throws at render time and unmounts the whole
// tree (blank screen). Pass it through from the Viewport iteration.
export interface ToastItemProps {
  toast: ToastPrimitive.Root.ToastObject;
  type?: "success" | "warning" | "error" | "info";
  title: string;
  description?: string;
  className?: string;
}

const TYPE_STYLES: Record<NonNullable<ToastItemProps["type"]>, {
  iconClass: string;
  borderClass: string;
  icon: typeof CheckCircle2;
}> = {
  success: {
    iconClass: "text-emerald-600 dark:text-emerald-400",
    borderClass: "border-l-emerald-500",
    icon: CheckCircle2,
  },
  warning: {
    iconClass: "text-amber-600 dark:text-amber-400",
    borderClass: "border-l-amber-500",
    icon: AlertCircle,
  },
  error: {
    iconClass: "text-destructive",
    borderClass: "border-l-destructive",
    icon: XCircle,
  },
  info: {
    iconClass: "text-sky-600 dark:text-sky-400",
    borderClass: "border-l-sky-500",
    icon: Info,
  },
};

export function ToastItem({ toast, type = "info", title, description, className }: ToastItemProps) {
  const { icon: Icon, iconClass, borderClass } = TYPE_STYLES[type];

  return (
    <ToastPrimitive.Root
      toast={toast}
      className={cn(
        "flex w-full items-start gap-3 rounded-lg border border-border border-l-4 bg-popover p-4 shadow-lg",
        borderClass,
        className,
      )}
    >
      <Icon className={cn("mt-0.5 h-5 w-5 shrink-0", iconClass)} />
      <div className="min-w-0 flex-1">
        <ToastPrimitive.Title className="text-sm font-semibold leading-tight">
          {title}
        </ToastPrimitive.Title>
        {description && (
          <ToastPrimitive.Description className="mt-1 text-xs leading-snug text-muted-foreground">
            {description}
          </ToastPrimitive.Description>
        )}
      </div>
      <ToastPrimitive.Close
        className="ml-1 rounded p-1 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        aria-label="Cerrar notificación"
      >
        <X className="h-4 w-4" />
      </ToastPrimitive.Close>
    </ToastPrimitive.Root>
  );
}

// ToastViewport renders the portal where all toasts mount. Mount
// ONCE inside the project's <ToastProvider> so it can read the
// toast store via useToastManager. The viewport position (bottom
// right) and the visual styling of each item (icon + color +
// border) are project conventions.
export function ToastViewport({ className }: { className?: string }) {
  const manager = ToastPrimitive.useToastManager();

  return (
    <ToastPrimitive.Portal>
      <ToastPrimitive.Viewport
        className={cn(
          "fixed bottom-4 right-4 z-50 flex w-full max-w-sm flex-col gap-2 outline-none",
          className,
        )}
      >
        {manager.toasts.map((toast) => (
          <ToastItem
            key={toast.id}
            toast={toast}
            type={(toast.type as ToastItemProps["type"]) ?? "info"}
            title={typeof toast.title === "string" ? toast.title : ""}
            description={
              typeof toast.description === "string" ? toast.description : undefined
            }
          />
        ))}
      </ToastPrimitive.Viewport>
    </ToastPrimitive.Portal>
  );
}

// Re-export Toast primitives so consumers can compose their own
// variants if needed. Most callers will only use useToast() from
// toast-context.tsx.
export { ToastPrimitive };
