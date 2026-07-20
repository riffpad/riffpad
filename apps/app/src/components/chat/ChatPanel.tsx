import { useEffect, useRef, useState } from "react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { AssistantMessage } from "./AssistantMessage";
import { FileChange } from "./FileChange";
import { ToolExecution } from "./ToolExecution";
import { UserMessage } from "./UserMessage";
import type { ChatItem } from "./types";
import type { Citation } from "./MarkdownRenderer";

interface ChatPanelProps {
  items: ChatItem[];
  emptyHint?: string;
  scrollRef?: React.RefObject<HTMLDivElement>;
  citations?: Citation[];
}

const NEAR_BOTTOM_THRESHOLD = 60;

export function ChatPanel({ items, emptyHint, scrollRef, citations }: ChatPanelProps) {
  const rootRef = useRef<HTMLDivElement | null>(null);
  const autoScrollRef = useRef(true);
  const [autoScroll, setAutoScroll] = useState(true);

  // Auto-scroll to bottom whenever new content arrives, but only if the user
  // is already near the bottom. If the user has scrolled up, we respect that
  // and stop auto-scrolling until they return to the bottom.
  useEffect(() => {
    if (!autoScrollRef.current) return;
    scrollRef?.current?.scrollIntoView({ behavior: "smooth" });
  }, [items, scrollRef]);

  useEffect(() => {
    const viewport = rootRef.current?.querySelector<HTMLDivElement>(
      "[data-radix-scroll-area-viewport]"
    );
    if (!viewport) return;

    const isNearBottom = () => {
      const distance =
        viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight;
      return distance < NEAR_BOTTOM_THRESHOLD;
    };

    const handleScroll = () => {
      const nearBottom = isNearBottom();
      if (nearBottom && !autoScrollRef.current) {
        autoScrollRef.current = true;
        setAutoScroll(true);
      } else if (!nearBottom && autoScrollRef.current) {
        autoScrollRef.current = false;
        setAutoScroll(false);
      }
    };

    viewport.addEventListener("scroll", handleScroll);
    return () => viewport.removeEventListener("scroll", handleScroll);
  }, []);

  return (
    <ScrollArea ref={rootRef} className="flex-1 px-3">
      <div className="space-y-5 py-4 px-1">
        {items.length === 0 && emptyHint && (
          <p className="text-xs text-mute">{emptyHint}</p>
        )}
        {items.map((item) => {
          switch (item.type) {
            case "user":
              return <UserMessage key={item.id} content={item.content} />;
            case "assistant":
              return (
                <AssistantMessage
                  key={item.id}
                  content={item.content}
                  reasoning={item.reasoning}
                  isStreaming={item.isStreaming}
                  citations={citations}
                />
              );
            case "tool":
              return (
                <ToolExecution
                  key={item.id}
                  toolName={item.toolName}
                  args={item.args}
                  result={item.result}
                  isError={item.isError}
                  isPartial={item.isPartial}
                />
              );
            case "file":
              return <FileChange key={item.id} path={item.path} />;
          }
        })}
        <div ref={scrollRef} />
      </div>
    </ScrollArea>
  );
}
