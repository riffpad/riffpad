import { Header } from "@/components/Header";
import { Hero } from "@/components/Hero";
import { Scene } from "@/components/Scene";
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
        {/* Screen-by-screen rhythm: odd screens are canvas (transparent, so the
            lattice shows through), even screens are flat surface-muted bands. */}
        <Hero />
        <div className="bg-surface-muted">
          <Scene />
        </div>
        <Architecture />
        <div className="bg-surface-muted">
          <HowItWorks />
        </div>
        <Security />
        <div className="bg-surface-muted">
          <FAQ />
        </div>
        <CTA />
      </main>
      <Footer />
    </>
  );
}
