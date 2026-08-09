"use client";

import { useState, type FormEvent } from "react";
import { useLanguage } from "./LanguageProvider";

const API_URL =
  process.env.NEXT_PUBLIC_RIFFPAD_API_URL ?? "https://api.riffpad.ai";
const WAITLIST_ENDPOINT = `${API_URL}/api/waitlist/subscribe`;

type Status = "idle" | "submitting" | "success" | "error";

export function WaitlistForm() {
  const { t } = useLanguage();
  const [status, setStatus] = useState<Status>("idle");

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const email = String(new FormData(form).get("email") ?? "").trim();
    if (!email || status === "submitting") return;

    setStatus("submitting");
    try {
      const res = await fetch(WAITLIST_ENDPOINT, {
        method: "POST",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({ email }),
      });
      if (!res.ok) throw new Error(`waitlist ${res.status}`);
      form.reset();
      setStatus("success");
    } catch {
      setStatus("error");
    }
  }

  if (status === "success") {
    return (
      <p className="mt-3 text-sm font-semibold text-accent">
        {t.hero.betaSuccess}
      </p>
    );
  }

  return (
    <div className="mt-3 flex flex-col items-center gap-2">
      <form
        onSubmit={handleSubmit}
        className="flex items-stretch gap-2"
        aria-label={t.hero.betaWaitlist}
      >
        <input
          type="email"
          name="email"
          required
          placeholder={t.hero.betaPlaceholder}
          disabled={status === "submitting"}
          className="h-10 w-56 border border-hairline-strong bg-surface px-3 font-mono text-sm text-ink outline-none transition-colors placeholder:text-mute focus:border-accent disabled:opacity-60 sm:w-64"
        />
        <button
          type="submit"
          disabled={status === "submitting"}
          aria-label={t.hero.betaWaitlist}
          className="flex h-10 w-10 shrink-0 cursor-pointer items-center justify-center bg-accent text-accent-ink transition-colors hover:bg-accent-hover disabled:opacity-60"
        >
          {status === "submitting" ? "…" : "→"}
        </button>
      </form>
      {status === "error" && (
        <p className="text-xs text-danger">{t.hero.betaError}</p>
      )}
    </div>
  );
}
