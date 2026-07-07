export default function Home() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center bg-neutral-950 px-6 text-center">
      <h1 className="text-5xl font-bold tracking-tight text-white sm:text-7xl">
        Riffpad
      </h1>
      <p className="mt-6 max-w-xl text-lg text-neutral-400">
        AI-Native code sketchbook. Capture inspiration on your phone, run
        prototypes in seconds, and bridge validated ideas to Cursor / Claude
        Code.
      </p>
      <div className="mt-10 flex gap-4">
        <a
          href="https://app.riffpad.ai"
          className="rounded-full bg-white px-6 py-3 font-medium text-neutral-950 transition hover:bg-neutral-200"
        >
          Get Started
        </a>
        <a
          href="/docs"
          className="rounded-full border border-neutral-700 px-6 py-3 font-medium text-white transition hover:border-neutral-500"
        >
          Read Docs
        </a>
      </div>
    </main>
  );
}
