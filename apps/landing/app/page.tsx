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
      <main>
        <Hero />
        <Architecture />
        <HowItWorks />
        <Security />
        <FAQ />
        <CTA />
      </main>
      <Footer />
    </>
  );
}
