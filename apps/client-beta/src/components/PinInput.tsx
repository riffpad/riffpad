import { useEffect, useRef } from "react";

interface Props {
  value: string;
  onChange(v: string): void;
  onComplete(v: string): void;
  disabled?: boolean;
  autoFocus?: boolean;
}

export default function PinInput({ value, onChange, onComplete, disabled, autoFocus }: Props) {
  const refs = useRef<Array<HTMLInputElement | null>>([]);

  useEffect(() => {
    if (autoFocus) refs.current[0]?.focus();
  }, [autoFocus]);

  function setVal(next: string) {
    const clean = next.toUpperCase().replace(/[^A-Z0-9]/g, "").slice(0, 6);
    onChange(clean);
    const idx = Math.min(clean.length, 5);
    refs.current[idx]?.focus();
    if (clean.length === 6) onComplete(clean);
  }

  return (
    <div id="pin-input" className="pin-input">
      {Array.from({ length: 6 }).map((_, i) => (
        <input
          key={i}
          ref={(el) => {
            refs.current[i] = el;
          }}
          className="pin-box"
          value={value[i] || ""}
          disabled={disabled}
          inputMode="text"
          autoCapitalize="characters"
          autoComplete="off"
          spellCheck={false}
          aria-label={`digit ${i + 1}`}
          onChange={(e) => {
            const v = e.target.value;
            if (v.length > 1) {
              // Autofill / direct fill of multiple chars behaves like paste.
              setVal(value.slice(0, i) + v);
              return;
            }
            if (!v) return;
            setVal(value.slice(0, i) + v.toUpperCase() + value.slice(i + 1));
          }}
          onKeyDown={(e) => {
            if (e.key === "Backspace" && !value[i] && i > 0) {
              refs.current[i - 1]?.focus();
            }
          }}
          onPaste={(e) => {
            e.preventDefault();
            setVal(value.slice(0, i) + e.clipboardData.getData("text"));
          }}
        />
      ))}
    </div>
  );
}
