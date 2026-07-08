import type { Metadata } from "next";
import localFont from "next/font/local";
import { IBM_Plex_Sans } from "next/font/google";
import "./globals.css";
import { LanguageProvider } from "@/components/LanguageProvider";
import { ThemeProvider } from "@/components/ThemeProvider";

const ibmPlexSans = IBM_Plex_Sans({
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
  variable: "--font-ibm-plex-sans",
  display: "swap",
});

const geistMono = localFont({
  src: "./fonts/GeistMonoVF.woff",
  variable: "--font-geist-mono",
  weight: "100 900",
});

export const metadata: Metadata = {
  metadataBase: new URL("https://riffpad.ai"),
  title: "Riffpad - The AI-native workspace in seconds.",
  description:
    "Brainstorm, capture ideas, and build light prototypes anywhere—on your phone, tablet, or desktop. Riffpad turns a sentence into a running workspace, no setup required.",
  keywords: [
    "AI coding",
    "code sketchbook",
    "prototype sandbox",
    "Cursor bridge",
    "mobile coding",
  ],
  alternates: {
    canonical: "/",
  },
  openGraph: {
    title: "Riffpad - The AI-native workspace in seconds.",
    description:
      "Brainstorm, capture ideas, and build light prototypes anywhere—on your phone, tablet, or desktop. Riffpad turns a sentence into a running workspace, no setup required.",
    url: "https://riffpad.ai",
    siteName: "Riffpad",
    locale: "en_US",
    type: "website",
    images: [
      {
        url: "/og.png?v=1",
        width: 1200,
        height: 630,
        alt: "Riffpad - The AI-native workspace in seconds.",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "Riffpad - The AI-native workspace in seconds.",
    description:
      "Brainstorm, capture ideas, and build light prototypes anywhere—on your phone, tablet, or desktop. Riffpad turns a sentence into a running workspace, no setup required.",
    images: ["/og.png?v=1"],
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
    <html lang="en">
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body
        className={`${ibmPlexSans.variable} ${geistMono.variable} antialiased`}
      >
        <ThemeProvider>
          <LanguageProvider>{children}</LanguageProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
