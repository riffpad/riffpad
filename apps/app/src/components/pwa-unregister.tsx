"use client";

import { useEffect } from "react";

export function PWAUnregister() {
  useEffect(() => {
    if (process.env.NODE_ENV !== "development") return;
    if (typeof navigator === "undefined" || !("serviceWorker" in navigator)) return;

    navigator.serviceWorker
      .getRegistrations()
      .then((registrations) => {
        for (const registration of registrations) {
          void registration.unregister();
        }
      })
      .catch(() => {
        // Ignore; service worker cleanup is best-effort in dev.
      });
  }, []);

  return null;
}
