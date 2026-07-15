"use client";

import { useState } from "react";
import { Sparkles } from "lucide-react";

export default function Home() {
  const [prompt, setPrompt] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: connect to /api/v1/workspaces and start a sandbox
    console.log("New spark:", prompt);
  };

  return (
    <main className="min-h-screen flex flex-col items-center justify-center px-4 sm:px-6">
      <div className="w-full max-w-2xl space-y-8 text-center">
        <div className="space-y-2">
          <h1 className="text-3xl sm:text-5xl font-bold tracking-tight">
            Riffpad
          </h1>
          <p className="text-muted-foreground text-base sm:text-lg">
            AI-native sketchbook for code. Describe an idea, make it run.
          </p>
        </div>

        <form onSubmit={handleSubmit} className="relative">
          <textarea
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="Describe your idea..."
            className="w-full min-h-[120px] rounded-xl border bg-background p-4 pr-14 text-base shadow-sm resize-none focus:outline-none focus:ring-2 focus:ring-ring"
          />
          <button
            type="submit"
            disabled={!prompt.trim()}
            className="absolute bottom-3 right-3 inline-flex items-center justify-center rounded-lg bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow transition-colors hover:bg-primary/90 disabled:opacity-50 disabled:pointer-events-none"
          >
            <Sparkles className="mr-1.5 h-4 w-4" />
            Spark
          </button>
        </form>

        <p className="text-xs text-muted-foreground">
          Press Enter to create a new workspace.
        </p>
      </div>
    </main>
  );
}
