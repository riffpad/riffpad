"use client";

import { memo } from "react";
import { useI18n } from "@/lib/i18n";

interface UserMessageProps {
  content: string;
}

function UserMessageImpl({ content }: UserMessageProps) {
  const { t } = useI18n();

  return (
    <div className="animate-fade-in-up" data-testid="user-message">
      <span className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-mute">
        {t("chat.user")}
      </span>
      <p className="text-sm leading-relaxed text-ink whitespace-pre-wrap">
        {content}
      </p>
    </div>
  );
}

export const UserMessage = memo(UserMessageImpl);
