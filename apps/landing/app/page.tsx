import { Header } from "@/components/Header";
import { Hero } from "@/components/Hero";
import { SaasWorkspaceMockup } from "@/components/mockups";
import { Features } from "@/components/Features";
import { Comparison } from "@/components/Comparison";
import { FAQ } from "@/components/FAQ";
import { Testimonials } from "@/components/Testimonials";
import { Footer } from "@/components/Footer";

const productSchema = {
  "@context": "https://schema.org",
  "@type": "SoftwareApplication",
  name: "Riffpad",
  applicationCategory: "DeveloperApplication",
  operatingSystem: "Web, iOS, Android",
  description:
    "AI-Native lightweight workspace. Capture inspiration, run prototypes in an isolated sandbox, then bridge validated ideas to downstream agents.",
  url: "https://riffpad.ai",
  offers: {
    "@type": "Offer",
    price: "0",
    priceCurrency: "USD",
  },
  aggregateRating: {
    "@type": "AggregateRating",
    ratingValue: "4.8",
    ratingCount: "128",
  },
};

export default function Home() {
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(productSchema) }}
      />
      <Header />
      <main className="bg-background">
        <Hero />
        <section className="px-4 pb-16 pt-4 sm:px-6 sm:pb-20 lg:px-8">
          <SaasWorkspaceMockup />
        </section>
        <Features />
        <Comparison />
        <Testimonials />
        <FAQ />
      </main>
      <Footer />
    </>
  );
}
