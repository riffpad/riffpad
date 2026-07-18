import { User } from "lucide-react";

interface UserMessageProps {
  content: string;
}

export function UserMessage({ content }: UserMessageProps) {
  return (
    <div className="flex justify-end gap-2">
      <div className="max-w-[85%] rounded-2xl rounded-tr-sm bg-card-soft px-4 py-2.5 text-sm text-ink shadow-sm">
        <p className="whitespace-pre-wrap leading-relaxed">{content}</p>
      </div>
      <div className="mt-1 h-5 w-5 shrink-0 rounded-full bg-primary/10 flex items-center justify-center">
        <User className="h-3 w-3 text-primary" />
      </div>
    </div>
  );
}
