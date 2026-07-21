"use client";

import { memo } from "react";
import { useI18n } from "@/lib/i18n";
import { MarkdownRenderer, type Citation } from "./MarkdownRenderer";
import { MessageActions } from "./MessageActions";
import { useSmoothTypewriter } from "./useSmoothTypewriter";

interface AssistantMessageProps {
  content: string;
  reasoning?: string;
  isStreaming?: boolean;
  citations?: Citation[];
  onRegenerate?: () => void;
}

function AssistantMessageImpl({
  content,
  reasoning,
  isStreaming,
  citations,
  onRegenerate,
}: AssistantMessageProps) {
  const { t } = useI18n();
  const displayed = useSmoothTypewriter(content, !!isStreaming);

  return (
    <div
      className="animate-fade-in-up"
      data-testid="assistant-message"
      data-streaming={isStreaming ? "true" : "false"}
    >
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
            <MarkdownRenderer content={isStreaming ? displayed : content} citations={citations} />
          </div>
        ) : null}
        {!isStreaming && content ? (
          <MessageActions content={content} onRegenerate={onRegenerate} />
        ) : null}
      </div>
    </div>
  );
}

export const AssistantMessage = memo(AssistantMessageImpl);
