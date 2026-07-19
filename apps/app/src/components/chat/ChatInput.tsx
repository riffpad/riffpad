"use client";

import { memo, useState } from "react";
import { ArrowUp } from "lucide-react";
import { Button } from "@/components/ui/button";
import { AutoResizeTextarea } from "./AutoResizeTextarea";
import { useI18n } from "@/lib/i18n";

interface ChatInputProps {
  onSend: (content: string) => void;
  disabled?: boolean;
}

function ChatInputImpl({ onSend, disabled }: ChatInputProps) {
  const { t } = useI18n();
  const [prompt, setPrompt] = useState("");

  const handleSubmit = (e: React.FormEvent | React.KeyboardEvent) => {
    e.preventDefault();
    if (!prompt.trim()) return;
    onSend(prompt.trim());
    setPrompt("");
  };

  return (
    <div className="px-3 pt-2 pb-3">
      <form onSubmit={handleSubmit} className="flex items-end gap-2 rounded-full border border-hairline bg-card px-3 py-2">
        <AutoResizeTextarea
          value={prompt}
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
          className="py-1.5"
        />
        <Button
          type="submit"
          disabled={!prompt.trim() || disabled}
          aria-label={t("prompt.send")}
          className="ml-auto flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary p-0 text-primary-foreground transition hover:opacity-90 disabled:opacity-50"
        >
          <ArrowUp className="h-4 w-4" />
        </Button>
      </form>
    </div>
  );
}

export const ChatInput = memo(ChatInputImpl);
