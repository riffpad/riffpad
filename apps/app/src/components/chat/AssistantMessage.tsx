"use client";

import { useI18n } from "@/lib/i18n";
import { MarkdownRenderer } from "./MarkdownRenderer";
import { useSmoothTypewriter } from "./useSmoothTypewriter";

interface AssistantMessageProps {
  content: string;
  reasoning?: string;
  isStreaming?: boolean;
}

export function AssistantMessage({
  content,
  reasoning,
  isStreaming,
}: AssistantMessageProps) {
  const { t } = useI18n();
  const displayed = useSmoothTypewriter(content, !!isStreaming);

  return (
    <div className="animate-fade-in-up">
      <span className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-mute">
        {t("chat.agent")}
      </span>
      <div className="space-y-2">
        {reasoning ? (
          <div className="text-xs italic text-mute leading-relaxed whitespace-pre-wrap border-l-2 border-hairline pl-2">
            {reasoning}
          </div>
        ) : null}
        {isStreaming ? (
          <div className="text-sm leading-relaxed text-body whitespace-pre-wrap">
            {displayed}
            <span className="ml-0.5 inline-block h-3.5 w-0.5 animate-pulse bg-primary align-middle" />
          </div>
        ) : content ? (
          <div className="text-sm leading-relaxed text-body">
            <MarkdownRenderer content={content} />
          </div>
        ) : null}
      </div>
    </div>
  );
}
