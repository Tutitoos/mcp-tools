// Shared modal for every detail/action view in the panel (tools,
// services, configure, models, settings, plugins). One place decides
// surface color, sizing, and header/footer layout — the per-route
// dialogs were four hand-rolled copies that had already drifted
// (footer present in 2 of 4, three different widths, log panes with
// the page background instead of an elevated surface).
//
// Sizes are content-width presets; height is owned by the content
// (JobLogPane caps itself). Every modal gets a footer: pass `footer`
// to replace the default "Cerrar" button, not to add one.

import type { ReactNode, Ref } from "react";

import { Button } from "~/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "~/components/ui/dialog";
import type { JobLine } from "~/lib/sse";
import { cn } from "~/lib/utils";

const SIZES = {
  sm: "sm:max-w-sm",
  md: "sm:max-w-lg",
  lg: "sm:max-w-2xl",
  xl: "sm:max-w-3xl",
} as const;

export type ModalSize = keyof typeof SIZES;

export function Modal({
  open,
  onOpenChange,
  title,
  description,
  size = "md",
  footer,
  children,
}: {
  open: boolean;
  onOpenChange: (next: boolean) => void;
  title: ReactNode;
  description?: ReactNode;
  size?: ModalSize;
  /** Replaces the default "Cerrar" button. */
  footer?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={SIZES[size]}>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          {description !== undefined && (
            <DialogDescription>{description}</DialogDescription>
          )}
        </DialogHeader>
        {children}
        <DialogFooter>
          {footer ?? (
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              Cerrar
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// JobLogPane is the terminal-style well used inside modals and the
// jobs/logs pages. The page-dark background (`bg-background`) is
// intentional: it reads as an inset terminal against the modal's
// elevated `bg-card` surface.
export function JobLogPane({
  lines,
  waiting = "Iniciando…",
  done = false,
  ok = false,
  error,
  okText = "completado",
  className,
  ref,
}: {
  lines: JobLine[];
  /** Placeholder while no output has arrived yet. */
  waiting?: string;
  done?: boolean;
  ok?: boolean;
  error?: string | null;
  okText?: string;
  className?: string;
  /** React 19 ref-as-prop; used by the logs page for follow-scroll. */
  ref?: Ref<HTMLDivElement>;
}) {
  return (
    <div
      ref={ref}
      className={cn(
        "max-h-80 overflow-y-auto rounded-md border border-border bg-background p-3 font-mono text-xs",
        className,
      )}
    >
      {lines.length === 0 && !done && (
        <p className="text-muted-foreground">{waiting}</p>
      )}
      {lines.map((l, i) => (
        <div
          key={i}
          className={
            l.stream === "stderr" ? "text-warning" : "text-foreground/90"
          }
        >
          {l.text}
        </div>
      ))}
      {done && (
        <div className={ok ? "mt-2 text-success" : "mt-2 text-destructive"}>
          {ok ? `✓ ${okText}` : `✗ ${error ?? "falló"}`}
        </div>
      )}
    </div>
  );
}
