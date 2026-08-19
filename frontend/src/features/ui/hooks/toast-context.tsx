import { createContext, useContext, useMemo, type ReactNode } from "react";
import { Toast } from "@base-ui/react/toast";

// ToastOptions is the wire shape accepted by useToast(). The
// `timeout` field controls auto-dismiss: positive ms to auto-close,
// 0 to require a manual X click. The `type` field is purely
// cosmetic — base-ui toasts carry no built-in type system, so we
// type-check the union here and the visual styling (icon, color,
// border) is applied in components/ui/toast.tsx via ToastItem.
export interface ToastOptions {
  title: string;
  description?: string;
  type?: "success" | "warning" | "error" | "info";
  timeout?: number;
}

// Auto-dismiss policy: success and info toasts close themselves
// after 5s; warning and error toasts stay until the user clicks
// the X. Callers can override by passing an explicit `timeout`.
const DEFAULT_TIMEOUT_BY_TYPE: Record<NonNullable<ToastOptions["type"]>, number> = {
  success: 5000,
  info: 5000,
  warning: 0,
  error: 0,
};

export interface ToastApi {
  success: (title: string, description?: string) => string;
  info: (title: string, description?: string) => string;
  warning: (title: string, description?: string) => string;
  error: (title: string, description?: string) => string;
  // Escape hatch for callers that need to pass an explicit timeout
  // or non-string content (e.g. JSX in the description).
  notify: (options: ToastOptions) => string;
}

const ToastContext = createContext<ToastApi | null>(null);

// ToastManagerBridge lives as a descendant of <Toast.Provider>
// (mounted by ToastProvider below). It is the only place that calls
// useToastManager, so the toast store context is already provided
// when this hook runs. It exposes the typed ToastApi to its own
// subtree via ToastContext.
//
// Splitting the wrapper in two — outer Provider that only mounts
// the third-party Provider, inner bridge that consumes its context
// — is required because any hook that reads from a Provider's
// context must be called from a descendant of that Provider. If we
// called useToastManager in the same component that mounts
// <Toast.Provider>, the context would be missing on the very first
// render and the hook would throw — blank screen.
function ToastManagerBridge({ children }: { children: ReactNode }) {
  const manager = Toast.useToastManager();

  const api = useMemo<ToastApi>(
    () => ({
      notify({ title, description, type = "info", timeout }) {
        return manager.add({
          title,
          description,
          type,
          timeout: timeout ?? DEFAULT_TIMEOUT_BY_TYPE[type],
        });
      },
      success(title, description) {
        return manager.add({
          title,
          description,
          type: "success",
          timeout: DEFAULT_TIMEOUT_BY_TYPE.success,
        });
      },
      info(title, description) {
        return manager.add({
          title,
          description,
          type: "info",
          timeout: DEFAULT_TIMEOUT_BY_TYPE.info,
        });
      },
      warning(title, description) {
        return manager.add({
          title,
          description,
          type: "warning",
          timeout: DEFAULT_TIMEOUT_BY_TYPE.warning,
        });
      },
      error(title, description) {
        return manager.add({
          title,
          description,
          type: "error",
          timeout: DEFAULT_TIMEOUT_BY_TYPE.error,
        });
      },
    }),
    [manager],
  );

  return <ToastContext.Provider value={api}>{children}</ToastContext.Provider>;
}

// ToastProvider is the outer wrapper that mounts base-ui's
// <Toast.Provider>. It contains no hooks of its own — every hook
// that depends on the toast store lives inside ToastManagerBridge.
// Mount once at the top of the app, above any consumer (typically
// in main.tsx next to the other context providers).
export function ToastProvider({ children }: { children: ReactNode }) {
  return (
    <Toast.Provider>
      <ToastManagerBridge>{children}</ToastManagerBridge>
    </Toast.Provider>
  );
}

// useToast returns the typed ToastApi. Must be called inside a
// ToastProvider subtree.
export function useToast(): ToastApi {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error("useToast must be used inside ToastProvider");
  }
  return ctx;
}
