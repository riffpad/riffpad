"use client";

import { memo, useCallback, useEffect, useRef, useState } from "react";
import { Brain, CornerDownLeft, Paperclip } from "lucide-react";
import { Button } from "@/components/ui/button";
import { AutoResizeTextarea } from "./AutoResizeTextarea";
import { useI18n } from "@/lib/i18n";

const MIN_HEIGHT = 40;
const MAX_HEIGHT = 200;

interface ChatInputProps {
  onSend: (content: string) => void;
  disabled?: boolean;
}

function ChatInputImpl({ onSend, disabled }: ChatInputProps) {
  const { t } = useI18n();
  const [prompt, setPrompt] = useState("");
  const [controlledHeight, setControlledHeight] = useState<number | undefined>(
    undefined
  );
  const draggingRef = useRef(false);
  const startYRef = useRef(0);
  const startHeightRef = useRef(0);

  const handleSubmit = (e: React.FormEvent | React.KeyboardEvent) => {
    e.preventDefault();
    if (!prompt.trim()) return;
    onSend(prompt.trim());
    setPrompt("");
    setControlledHeight(undefined);
  };

  const handleResizeStart = (e: React.MouseEvent | React.TouchEvent) => {
    const clientY = "touches" in e ? e.touches[0].clientY : e.clientY;
    draggingRef.current = true;
    startYRef.current = clientY;
    startHeightRef.current = controlledHeight ?? MIN_HEIGHT;
    e.preventDefault();
  };

  const handleResizeMove = useCallback((clientY: number) => {
    if (!draggingRef.current) return;
    const delta = startYRef.current - clientY;
    const nextHeight = Math.min(
      Math.max(startHeightRef.current + delta, MIN_HEIGHT),
      MAX_HEIGHT
    );
    setControlledHeight(nextHeight);
  }, []);

  const handleResizeEnd = useCallback(() => {
    draggingRef.current = false;
  }, []);

  useEffect(() => {
    const onMouseMove = (e: MouseEvent) => handleResizeMove(e.clientY);
    const onTouchMove = (e: TouchEvent) =>
      handleResizeMove(e.touches[0].clientY);
    const onUp = () => handleResizeEnd();

    window.addEventListener("mousemove", onMouseMove);
    window.addEventListener("mouseup", onUp);
    window.addEventListener("touchmove", onTouchMove);
    window.addEventListener("touchend", onUp);
    return () => {
      window.removeEventListener("mousemove", onMouseMove);
      window.removeEventListener("mouseup", onUp);
      window.removeEventListener("touchmove", onTouchMove);
      window.removeEventListener("touchend", onUp);
    };
  }, [handleResizeMove, handleResizeEnd]);

  return (
    <div className="relative px-3 pt-1 pb-2 mb-2">
      {/* Invisible top-edge resize handle */}
      <div
        onMouseDown={handleResizeStart}
        onTouchStart={handleResizeStart}
        className="absolute -top-1 left-0 right-0 z-10 h-2 cursor-ns-resize"
        aria-hidden="true"
      />
      <form
        onSubmit={handleSubmit}
        className="flex flex-col rounded-none border border-hairline bg-card px-3 pt-2 pb-1.5 focus-within:border-primary focus-within:ring-1 focus-within:ring-primary transition-shadow"
      >
        <AutoResizeTextarea
          value={prompt}
          controlledHeight={controlledHeight}
          onChange={(e) => setPrompt(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              if (prompt.trim()) {
                handleSubmit(e);
              }
            }
          }}
          placeholder={t("chat.placeholder")}
          disabled={disabled}
          minRows={1}
          maxRows={10}
          className="py-1 text-sm"
        />

        {/* Bottom toolbar */}
        <div className="mt-1 flex items-center justify-between pt-0.5">
          <div className="flex items-center gap-1">
            <Button
              type="button"
              variant="ghost"
              size="icon"
              disabled={disabled}
              aria-label="Switch model"
              className="h-7 w-7 rounded-md text-mute hover:text-ink hover:bg-card-soft"
            >
              <Brain className="h-4 w-4" />
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              disabled={disabled}
              aria-label="Upload file"
              className="h-7 w-7 rounded-md text-mute hover:text-ink hover:bg-card-soft"
            >
              <Paperclip className="h-4 w-4" />
            </Button>
          </div>

          <Button
            type="submit"
            disabled={!prompt.trim() || disabled}
            aria-label={t("prompt.send")}
            className="h-7 w-7 shrink-0 rounded-md border border-hairline bg-transparent p-0 text-mute transition hover:border-ash hover:text-ink hover:bg-transparent disabled:opacity-50"
          >
            <CornerDownLeft className="h-4 w-4" />
          </Button>
        </div>
      </form>
    </div>
  );
}

export const ChatInput = memo(ChatInputImpl);
