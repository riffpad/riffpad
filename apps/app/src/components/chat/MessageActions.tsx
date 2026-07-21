"use client";

import { memo, useState } from "react";
import { Check, Copy, MoreHorizontal, RefreshCw } from "lucide-react";
import { useI18n } from "@/lib/i18n";

interface MessageActionsProps {
  content: string;
  onRegenerate?: () => void;
}

function MessageActionsImpl({ content, onRegenerate }: MessageActionsProps) {
  const { t } = useI18n();
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // ignore
    }
  };

  return (
    <div className="mt-2 flex items-center gap-0.5">
      <ActionButton
        onClick={handleCopy}
        label={t("chat.copy")}
        icon={copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
      />
      {onRegenerate ? (
        <ActionButton
          onClick={onRegenerate}
          label={t("chat.regenerate")}
          icon={<RefreshCw className="h-3.5 w-3.5" />}
        />
      ) : null}
      <ActionButton
        label={t("chat.more")}
        icon={<MoreHorizontal className="h-3.5 w-3.5" />}
      />
    </div>
  );
}

function ActionButton({
  onClick,
  label,
  icon,
}: {
  onClick?: () => void;
  label: string;
  icon: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex h-7 w-7 items-center justify-center rounded-md text-mute transition hover:bg-card-soft hover:text-ink"
      aria-label={label}
      title={label}
    >
      {icon}
    </button>
  );
}

export const MessageActions = memo(MessageActionsImpl);
