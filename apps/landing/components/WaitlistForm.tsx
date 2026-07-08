"use client";

import { useForm, ValidationError } from "@formspree/react";
import { useLanguage } from "./LanguageProvider";

export function WaitlistForm({ center = false }: { center?: boolean }) {
  const { t } = useLanguage();
  const [state, handleSubmit] = useForm("xjgqddar");

  if (state.succeeded) {
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
      <div className="flex-1">
        <input
          type="email"
          name="email"
          id="email"
          required
          placeholder={t.waitlist.placeholder}
          className="w-full rounded-md border border-hairline bg-surface px-4 py-3 text-base text-foreground outline-none transition placeholder:text-muted focus:border-accent"
        />
        <ValidationError
          prefix="Email"
          field="email"
          errors={state.errors}
          className="mt-1 text-sm text-accent-red"
        />
      </div>
      <button
        type="submit"
        disabled={state.submitting}
        className="rounded-md bg-accent px-6 py-3 text-base font-bold text-on-accent transition hover:bg-accent-pressed disabled:opacity-60"
      >
        {state.submitting ? "..." : t.waitlist.cta}
      </button>
    </form>
  );
}
