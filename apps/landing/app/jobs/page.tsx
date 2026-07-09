"use client";

import { Header } from "@/components/Header";
import { Footer } from "@/components/Footer";
import { useLanguage } from "@/components/LanguageProvider";

export default function JobsPage() {
  const { t } = useLanguage();
  const p = t.pages.jobs;

  return (
    <>
      <Header />
      <main className="min-h-[60vh] bg-background px-4 py-20 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-3xl">
          <h1 className="text-3xl font-bold text-foreground sm:text-4xl">{p.title}</h1>
          <p className="mt-4 text-lg text-body">{p.description}</p>

          <div className="mt-10 rounded-md border border-hairline bg-surface p-6">
            <p className="font-semibold text-foreground">{p.noRoles}</p>
            <p className="mt-2 text-body">{p.contact}</p>
          </div>
        </div>
      </main>
      <Footer />
    </>
  );
}
