"use client";

import { Header } from "@/components/Header";
import { Footer } from "@/components/Footer";
import { useLanguage } from "@/components/LanguageProvider";

export default function AboutPage() {
  const { t } = useLanguage();
  const p = t.pages.about;

  return (
    <>
      <Header />
      <main className="min-h-[60vh] bg-background px-4 py-20 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-3xl">
          <h1 className="text-3xl font-bold text-foreground sm:text-4xl">{p.title}</h1>
          <p className="mt-4 text-lg text-body">{p.description}</p>

          <div className="mt-10">
            <h2 className="text-xl font-bold text-foreground">Mission</h2>
            <p className="mt-3 text-body">{p.mission}</p>
          </div>

          <div className="mt-10">
            <h2 className="text-xl font-bold text-foreground">{p.team}</h2>
            <p className="mt-3 text-body">{p.teamText}</p>
          </div>
        </div>
      </main>
      <Footer />
    </>
  );
}
