import type { Metadata } from "next";
import localFont from "next/font/local";
import { IBM_Plex_Sans } from "next/font/google";
import "./globals.css";
import { LanguageProvider } from "@/components/LanguageProvider";

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
  title: "Riffpad - AI-Native Code Sketchbook",
  description:
    "Capture code inspiration anywhere, run prototypes in an isolated sandbox in under a second, and bridge validated ideas to Cursor / Claude Code.",
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
    title: "Riffpad - AI-Native Code Sketchbook",
    description:
      "Capture code inspiration anywhere, run prototypes in an isolated sandbox in under a second, and bridge validated ideas to Cursor / Claude Code.",
    url: "https://riffpad.ai",
    siteName: "Riffpad",
    locale: "en_US",
    type: "website",
    images: [
      {
        url: "/og.png",
        width: 1200,
        height: 630,
        alt: "Riffpad - AI-Native Code Sketchbook",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "Riffpad - AI-Native Code Sketchbook",
    description:
      "Capture code inspiration anywhere, run prototypes in an isolated sandbox in under a second, and bridge validated ideas to Cursor / Claude Code.",
    images: ["/og.png"],
  },
  robots: {
    index: true,
    follow: true,
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className={`${ibmPlexSans.variable} ${geistMono.variable} antialiased`}
      >
        <LanguageProvider>{children}</LanguageProvider>
      </body>
    </html>
  );
}
