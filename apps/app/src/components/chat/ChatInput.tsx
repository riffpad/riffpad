"use client";

import { memo, useState } from "react";
import { Send } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
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
    <div className="p-3 border-t border-hairline bg-card/80 backdrop-blur">
      <form onSubmit={handleSubmit} className="flex items-center gap-2 rounded-full border border-hairline bg-card px-3 py-2 shadow-sm">
        <Textarea
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
          rows={1}
          disabled={disabled}
          className="flex-1 resize-none bg-transparent border-0 text-ink placeholder:text-ash focus-visible:ring-0 focus-visible:ring-offset-0 min-h-0 py-1 px-0 max-h-32"
        />
        <Button
          type="submit"
          disabled={!prompt.trim() || disabled}
          aria-label={t("prompt.send")}
          className="h-8 w-8 shrink-0 rounded-full bg-primary text-primary-foreground hover:bg-primary-pressed disabled:opacity-50 p-0"
        >
          <Send className="h-4 w-4" />
        </Button>
      </form>
    </div>
  );
}

export const ChatInput = memo(ChatInputImpl);
