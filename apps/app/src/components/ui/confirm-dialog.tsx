"use client";

import { useEffect, useRef } from "react";

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  message: string;
  confirmLabel: string;
  cancelLabel: string;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel,
  cancelLabel,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const cancelRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!open) return;
    // Destructive actions default focus to cancel.
    cancelRef.current?.focus();
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onCancel();
    };
    document.addEventListener("keydown", handler);
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", handler);
      document.body.style.overflow = "";
    };
  }, [open, onCancel]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-[100] flex items-center justify-center p-4"
      onClick={onCancel}
    >
      <div className="absolute inset-0 bg-ink/30 backdrop-blur-sm" />
      <div
        role="alertdialog"
        aria-modal="true"
        aria-label={title}
        className="relative w-full max-w-sm rounded-xl border border-hairline bg-card p-5 shadow-2xl animate-fade-in-up"
        onClick={(e) => e.stopPropagation()}
      >
        <h3 className="text-sm font-bold text-ink">{title}</h3>
        <p className="mt-2 text-sm text-body leading-relaxed">{message}</p>
        <div className="mt-5 flex justify-end gap-2">
          <button
            ref={cancelRef}
            onClick={onCancel}
            className="h-8 rounded-md px-3 text-xs font-semibold text-body transition hover:bg-card-soft hover:text-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
          >
            {cancelLabel}
          </button>
          <button
            onClick={onConfirm}
            className="h-8 rounded-md bg-accent-red px-3 text-xs font-bold text-white transition hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-red/60"
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
