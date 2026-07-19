"use client";

import { memo } from "react";
import { useI18n } from "@/lib/i18n";
import { MarkdownRenderer } from "./MarkdownRenderer";
import { useSmoothTypewriter } from "./useSmoothTypewriter";

interface AssistantMessageProps {
  content: string;
  reasoning?: string;
  isStreaming?: boolean;
}

function AssistantMessageImpl({
  content,
  reasoning,
  isStreaming,
}: AssistantMessageProps) {
  const { t } = useI18n();
  const displayed = useSmoothTypewriter(content, !!isStreaming);

  return (
    <div className="animate-fade-in-up" data-testid="assistant-message">
      <span className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-mute">
        {t("chat.agent")}
      </span>
      <div className="space-y-2" data-testid="assistant-content">
        {reasoning ? (
          <div className="text-xs italic text-mute leading-relaxed whitespace-pre-wrap border-l-2 border-hairline pl-2">
            {reasoning}
          </div>
        ) : null}
        {content ? (
          <div className="text-sm leading-relaxed text-body">
            <MarkdownRenderer content={isStreaming ? displayed : content} />
          </div>
        ) : null}
      </div>
    </div>
  );
}

export const AssistantMessage = memo(AssistantMessageImpl);
