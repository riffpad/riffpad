import type { Metadata } from "next";
import localFont from "next/font/local";
import type { ReactNode } from "react";
import "./globals.css";
import { ThemeProvider } from "@/components/ThemeProvider";
import { LanguageProvider } from "@/components/LanguageProvider";

const geistMono = localFont({
  src: "./fonts/GeistMonoVF.woff",
  variable: "--font-geist-mono",
  weight: "100 900",
  display: "swap",
});

export const metadata: Metadata = {
  metadataBase: new URL("https://riffpad.ai"),
  title: "Riffpad - Watch, approve, and steer AI coding agents from your phone",
  description:
    "Watch, approve and steer Claude Code, Codex and other AI coding CLIs from your phone. Local daemon, end-to-end encryption, zero-knowledge relay.",
  keywords: [
    "AI coding agent",
    "Claude Code",
    "Codex",
    "mobile remote",
    "approval",
    "E2EE",
    "Riffpad",
  ],
  alternates: {
    canonical: "/",
  },
  openGraph: {
    title: "Riffpad - Watch, approve, and steer AI coding agents from your phone",
    description:
      "Watch, approve and steer AI coding CLIs from your phone. Local daemon, end-to-end encryption, zero-knowledge relay.",
    url: "https://riffpad.ai",
    siteName: "Riffpad",
    locale: "en_US",
    type: "website",
    images: ["https://riffpad.ai/og.png"],
  },
  twitter: {
    card: "summary_large_image",
    title: "Riffpad - Watch, approve, and steer AI coding agents from your phone",
    description:
      "Watch, approve and steer AI coding CLIs from your phone. Local daemon, end-to-end encryption, zero-knowledge relay.",
    images: ["https://riffpad.ai/og.png"],
  },
  robots: {
    index: true,
    follow: true,
  },
};

const themeScript = `
  (function () {
    try {
      const stored = localStorage.getItem('riffpad-theme');
      const theme = stored === 'light' || stored === 'dark'
        ? stored
        : (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
      document.documentElement.dataset.theme = theme;
    } catch (e) {}
  })();
`;

const faviconScript = `
  (function () {
    var link = document.getElementById("riffpad-favicon");
    if (!link) return;
    var html = document.documentElement;
    function currentTheme() {
      var stored = null;
      try { stored = localStorage.getItem("riffpad-theme"); } catch (e) {}
      var attr = html.getAttribute("data-theme");
      if (attr === "dark" || stored === "dark") return "dark";
      if (attr === "light" || stored === "light") return "light";
      return window.matchMedia("(prefers-color-scheme: dark)").matches
        ? "dark"
        : "light";
    }
    function apply() {
      link.setAttribute(
        "href",
        currentTheme() === "dark" ? "/favicon-dark.png" : "/favicon-light.png"
      );
    }
    apply();
    if ("MutationObserver" in window) {
      new MutationObserver(apply).observe(html, {
        attributes: true,
        attributeFilter: ["data-theme"],
      });
    }
  })();
`;

export default function RootLayout({
  children,
}: Readonly<{
  children: ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
        <link
          rel="icon"
          type="image/png"
          href="/favicon-light.png"
          id="riffpad-favicon"
        />
        <script dangerouslySetInnerHTML={{ __html: faviconScript }} />
      </head>
      <body className={`${geistMono.variable} antialiased`}>
        <ThemeProvider>
          <LanguageProvider>{children}</LanguageProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
