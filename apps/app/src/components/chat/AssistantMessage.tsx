import { Bot } from "lucide-react";

interface AssistantMessageProps {
  content: string;
  reasoning?: string;
  isStreaming?: boolean;
}

export function AssistantMessage({ content, reasoning, isStreaming }: AssistantMessageProps) {
  return (
    <div className="flex gap-2">
      <div className="mt-1 h-5 w-5 shrink-0 rounded-full bg-accent-blue/10 flex items-center justify-center">
        <Bot className="h-3 w-3 text-accent-blue" />
      </div>
      <div className="max-w-[90%] min-w-0 space-y-2">
        {reasoning ? (
          <div className="text-xs italic text-mute leading-relaxed whitespace-pre-wrap border-l-2 border-hairline pl-2">
            {reasoning}
          </div>
        ) : null}
        {content ? (
          <div className="text-sm text-body leading-relaxed whitespace-pre-wrap">
            {content}
          </div>
        ) : isStreaming && !reasoning ? (
          <div className="flex items-center gap-2 text-xs text-mute">
            <span className="inline-block h-1.5 w-1.5 rounded-full bg-accent-blue animate-pulse" />
            Working...
          </div>
        ) : null}
      </div>
    </div>
  );
}
