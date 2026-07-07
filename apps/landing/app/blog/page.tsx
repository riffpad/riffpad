import { Header } from "@/components/Header";
import { Footer } from "@/components/Footer";

export const metadata = {
  title: "Blog - Riffpad",
  description: "Stories, updates, and engineering notes from the Riffpad team.",
};

export default function BlogPage() {
  return (
    <>
      <Header />
      <main className="bg-background px-4 py-20 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-3xl text-center">
          <h1 className="text-3xl font-bold text-foreground sm:text-4xl">Blog</h1>
          <p className="mt-4 text-body">Coming soon.</p>
        </div>
      </main>
      <Footer />
    </>
  );
}
