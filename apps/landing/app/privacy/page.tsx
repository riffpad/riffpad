"use client";

import { Header } from "@/components/Header";
import { Footer } from "@/components/Footer";
import { useLanguage } from "@/components/LanguageProvider";

export default function PrivacyPage() {
  const { t } = useLanguage();
  const p = t.pages.privacy;

  return (
    <>
      <Header />
      <main className="min-h-[60vh] bg-background px-4 py-20 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-3xl">
          <h1 className="text-3xl font-bold text-foreground sm:text-4xl">{p.title}</h1>
          <p className="mt-4 text-lg text-body">{p.description}</p>
          <p className="mt-2 text-sm text-muted">{p.lastUpdated}</p>

          <div className="mt-10 space-y-8">
            {p.sections.map((section) => (
              <div key={section.title}>
                <h2 className="text-xl font-bold text-foreground">{section.title}</h2>
                <p className="mt-3 text-body">{section.content}</p>
              </div>
            ))}
          </div>
        </div>
      </main>
      <Footer />
    </>
  );
}
