"use client";

import { Header } from "@/components/Header";
import { Footer } from "@/components/Footer";
import { useLanguage } from "@/components/LanguageProvider";

export default function ChangelogPage() {
  const { t } = useLanguage();
  const p = t.pages.changelog;

  return (
    <>
      <Header />
      <main className="min-h-[60vh] bg-background px-4 py-20 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-3xl">
          <h1 className="text-3xl font-bold text-foreground sm:text-4xl">{p.title}</h1>
          <p className="mt-4 text-lg text-body">{p.description}</p>

          <div className="mt-10 space-y-6">
            {p.items.map((item) => (
              <div
                key={item.version}
                className="rounded-md border border-hairline bg-surface p-6"
              >
                <div className="flex flex-wrap items-center gap-3">
                  <span className="text-lg font-bold text-foreground">{item.version}</span>
                  <span className="text-sm text-muted">{item.date}</span>
                </div>
                <p className="mt-2 text-body">{item.content}</p>
              </div>
            ))}
          </div>
        </div>
      </main>
      <Footer />
    </>
  );
}
