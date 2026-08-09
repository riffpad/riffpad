"use client";

import { useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useLanguage } from "@/components/LanguageProvider";

const API_URL =
  process.env.NEXT_PUBLIC_RIFFPAD_API_URL ?? "https://api.riffpad.ai";

type Status = "idle" | "submitting" | "success" | "error";

export function UnsubscribeClient() {
  const { t } = useLanguage();
  const params = useSearchParams();
  const email = params.get("email") ?? "";
  const token = params.get("token") ?? "";
  const valid = email !== "" && token !== "";
  const [status, setStatus] = useState<Status>("idle");

  useEffect(() => {
    setStatus("idle");
  }, [email, token]);

  async function handleUnsubscribe() {
    if (!valid || status === "submitting") return;
    setStatus("submitting");
    try {
      const res = await fetch(`${API_URL}/api/waitlist/unsubscribe`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, token }),
      });
      if (!res.ok) throw new Error(`unsubscribe ${res.status}`);
      setStatus("success");
    } catch {
      setStatus("error");
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center px-6 py-16">
      <div className="w-full max-w-md border border-hairline-strong bg-surface p-8 shadow-card">
        <p className="label">{t.unsubscribe.label}</p>

        {!valid ? (
          <div className="mt-6">
            <h1 className="text-lg font-bold text-ink">{t.unsubscribe.invalidTitle}</h1>
            <p className="mt-2 text-sm leading-relaxed text-body">
              {t.unsubscribe.invalidDesc}
            </p>
          </div>
        ) : status === "success" ? (
          <div className="mt-6">
            <h1 className="text-lg font-bold text-accent">
              {t.unsubscribe.successTitle}
            </h1>
            <p className="mt-2 text-sm leading-relaxed text-body">
              {t.unsubscribe.successDesc}
            </p>
          </div>
        ) : (
          <>
            <h1 className="mt-6 text-lg font-bold text-ink">
              {t.unsubscribe.title}
            </h1>
            <p className="mt-2 text-sm leading-relaxed text-body">
              {t.unsubscribe.desc}
            </p>

            <label className="label mt-6 block">{t.unsubscribe.emailLabel}</label>
            <p className="mt-1 truncate border border-hairline bg-surface-muted px-3 py-2 text-sm text-ink">
              {email}
            </p>

            <button
              type="button"
              onClick={handleUnsubscribe}
              disabled={status === "submitting"}
              className="btn btn-primary mt-6 w-full"
            >
              {status === "submitting" ? "…" : t.unsubscribe.confirm}
            </button>

            {status === "error" && (
              <p className="mt-3 text-xs text-danger">{t.unsubscribe.error}</p>
            )}
          </>
        )}

        <a
          href="/"
          className="btn btn-ghost mt-8 -ml-2 px-2 text-xs"
        >
          {t.unsubscribe.back}
        </a>
      </div>
    </main>
  );
}
