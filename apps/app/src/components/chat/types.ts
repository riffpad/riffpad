export interface ChatUserItem {
  type: "user";
  id: string;
  content: string;
  timestamp: number;
}

export interface ChatAssistantItem {
  type: "assistant";
  id: string;
  content: string;
  reasoning?: string;
  isStreaming?: boolean;
  timestamp: number;
}

export interface ChatToolItem {
  type: "tool";
  id: string;
  toolName: string;
  args: unknown;
  result?: unknown;
  isError?: boolean;
  isPartial?: boolean;
  timestamp: number;
}

export interface ChatFileItem {
  type: "file";
  id: string;
  path: string;
  timestamp: number;
}

export type ChatItem =
  | ChatUserItem
  | ChatAssistantItem
  | ChatToolItem
  | ChatFileItem;
