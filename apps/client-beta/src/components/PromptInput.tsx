import { useI18n } from "../lib/i18n";

function StopIcon() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <rect x="5" y="5" width="14" height="14" />
    </svg>
  );
}

interface Props {
  running: boolean;
  ended: boolean;
  prompt: string;
  setPrompt(v: string): void;
  onSend(): void;
  onInterrupt(): void;
}

// The compose row under the timeline: prompt box, send button and the
// stop/interrupt control that appears while the agent is working (#300).
export default function PromptInput({ running, ended, prompt, setPrompt, onSend, onInterrupt }: Props) {
  const { t } = useI18n();
  return (
    <form
      className="prompt-form"
      autoComplete="off"
      onSubmit={(e) => {
        e.preventDefault();
        onSend();
      }}
    >
      <div className="prompt-wrap">
        <input
          placeholder={t("prompt_ph")}
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          name="message"
          autoComplete="off"
          inputMode="text"
          enterKeyHint="send"
          data-form-type="other"
          disabled={ended}
        />
        {prompt.trim() && !running && !ended && (
          <button type="submit" id="send-btn" className="send-btn" aria-label={t("send")}>→</button>
        )}
    </div>
      {running ? (
        <button type="button" className="danger interrupt" onClick={onInterrupt}>
          <StopIcon />
          {t("interrupt")}
        </button>
      ) : null}
    </form>
  );
}
