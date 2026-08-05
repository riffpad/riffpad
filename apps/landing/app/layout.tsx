import type { Metadata } from "next";
import localFont from "next/font/local";
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
  title: "Riffpad — The pocket remote for your AI coding agents",
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
    title: "Riffpad — The pocket remote for your AI coding agents",
    description:
      "Watch, approve and steer AI coding CLIs from your phone. Local daemon, end-to-end encryption, zero-knowledge relay.",
    url: "https://riffpad.ai",
    siteName: "Riffpad",
    locale: "en_US",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "Riffpad — The pocket remote for your AI coding agents",
    description:
      "Watch, approve and steer AI coding CLIs from your phone. Local daemon, end-to-end encryption, zero-knowledge relay.",
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

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body className={`${geistMono.variable} antialiased`}>
        <ThemeProvider>
          <LanguageProvider>{children}</LanguageProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
