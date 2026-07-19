"use client";

import { useCallback, useEffect, useRef } from "react";
import { cn } from "@/lib/utils";

export interface AutoResizeTextareaProps
  extends Omit<React.TextareaHTMLAttributes<HTMLTextAreaElement>, "ref"> {
  minRows?: number;
  maxRows?: number;
}

const LINE_HEIGHT = 20; // matches text-sm leading-relaxed ~20px per row
const MIN_HEIGHT = 24;
const MAX_HEIGHT = 200;

export function AutoResizeTextarea({
  minRows = 1,
  maxRows = 10,
  className,
  onInput,
  ...props
}: AutoResizeTextareaProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const adjustHeight = useCallback(
    (textarea: HTMLTextAreaElement) => {
      textarea.style.height = "auto";
      const targetHeight = Math.min(
        Math.max(textarea.scrollHeight, MIN_HEIGHT, minRows * LINE_HEIGHT),
        maxRows * LINE_HEIGHT,
        MAX_HEIGHT
      );
      textarea.style.height = `${targetHeight}px`;
    },
    [minRows, maxRows]
  );

  useEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;
    adjustHeight(textarea);
  }, [adjustHeight, props.value, props.defaultValue]);

  const handleInput = (e: React.FormEvent<HTMLTextAreaElement>) => {
    adjustHeight(e.currentTarget);
    onInput?.(e);
  };

  return (
    <textarea
      ref={textareaRef}
      onInput={handleInput}
      className={cn(
        "flex-1 resize-y bg-transparent border-0 text-ink placeholder:text-ash focus-visible:outline-none focus-visible:ring-0 focus-visible:ring-offset-0 min-h-0 py-1.5 px-0 overflow-auto",
        className
      )}
      style={{
        minHeight: `${Math.max(MIN_HEIGHT, minRows * LINE_HEIGHT)}px`,
        maxHeight: `${Math.min(maxRows * LINE_HEIGHT, MAX_HEIGHT)}px`,
      }}
      {...props}
    />
  );
}
