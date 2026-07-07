"use client";

import { useState } from "react";
import { useLanguage } from "./LanguageProvider";

export function WaitlistForm({ center = false }: { center?: boolean }) {
  const { t } = useLanguage();
  const [email, setEmail] = useState("");
  const [submitted, setSubmitted] = useState(false);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (email.trim()) {
      setSubmitted(true);
    }
  };

  if (submitted) {
    return (
      <p className={`text-base font-semibold text-accent ${center ? "text-center" : ""}`}>
        {t.waitlist.success}
      </p>
    );
  }

  return (
    <form
      onSubmit={handleSubmit}
      className={`flex flex-col gap-3 sm:flex-row ${center ? "justify-center" : ""}`}
    >
      <input
        type="email"
        required
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        placeholder={t.waitlist.placeholder}
        className="flex-1 rounded-md border border-hairline bg-surface px-4 py-3 text-base text-foreground outline-none transition placeholder:text-muted focus:border-accent"
      />
      <button
        type="submit"
        className="rounded-md bg-accent px-6 py-3 text-base font-bold text-on-accent transition hover:bg-accent-pressed"
      >
        {t.waitlist.cta}
      </button>
    </form>
  );
}
