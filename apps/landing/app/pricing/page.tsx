import { Header } from "@/components/Header";
import { Pricing } from "@/components/Pricing";
import { Footer } from "@/components/Footer";

export const metadata = {
  title: "Pricing - Riffpad",
  description: "Start free, pay only when it sticks. Riffpad pricing plans for indie hackers, creators, and teams.",
};

export default function PricingPage() {
  return (
    <>
      <Header />
      <main className="bg-background pt-8">
        <Pricing />
      </main>
      <Footer />
    </>
  );
}
