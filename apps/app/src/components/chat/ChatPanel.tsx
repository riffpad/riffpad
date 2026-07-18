import { ScrollArea } from "@/components/ui/scroll-area";
import { AssistantMessage } from "./AssistantMessage";
import { FileChange } from "./FileChange";
import { ToolExecution } from "./ToolExecution";
import { UserMessage } from "./UserMessage";
import type { ChatItem } from "./types";

interface ChatPanelProps {
  items: ChatItem[];
  emptyHint?: string;
  scrollRef?: React.RefObject<HTMLDivElement>;
}

export function ChatPanel({ items, emptyHint, scrollRef }: ChatPanelProps) {
  return (
    <ScrollArea className="flex-1 px-3">
      <div className="space-y-3 py-3">
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
                  isStreaming={item.isStreaming}
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
