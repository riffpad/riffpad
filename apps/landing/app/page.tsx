import { Header } from "@/components/Header";
import { Hero } from "@/components/Hero";
import { Architecture } from "@/components/Architecture";
import { HowItWorks } from "@/components/HowItWorks";
import { Security } from "@/components/Security";
import { FAQ } from "@/components/FAQ";
import { CTA } from "@/components/CTA";
import { Footer } from "@/components/Footer";

export default function Home() {
  return (
    <>
      <Header />
      <main className="relative">
        <Hero />
        <div className="bg-surface-muted">
          <Architecture />
        </div>
        <HowItWorks />
        <div className="bg-surface-muted">
          <Security />
        </div>
        <FAQ />
        <div className="bg-surface-muted">
          <CTA />
        </div>
      </main>
      <Footer />
    </>
  );
}
