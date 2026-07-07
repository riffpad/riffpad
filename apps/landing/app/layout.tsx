import type { Metadata } from "next";
import localFont from "next/font/local";
import "./globals.css";

const geistSans = localFont({
  src: "./fonts/GeistVF.woff",
  variable: "--font-geist-sans",
  weight: "100 900",
});
const geistMono = localFont({
  src: "./fonts/GeistMonoVF.woff",
  variable: "--font-geist-mono",
  weight: "100 900",
});

export const metadata: Metadata = {
  metadataBase: new URL("https://riffpad.ai"),
  title: "Riffpad - AI 时代的代码灵感草稿本",
  description:
    "随时随地捕捉代码灵感，秒级启动隔离沙箱运行原型，一键桥接到 Cursor / Claude Code。",
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
    title: "Riffpad - AI 时代的代码灵感草稿本",
    description:
      "随时随地捕捉代码灵感，秒级启动隔离沙箱运行原型，一键桥接到 Cursor / Claude Code。",
    url: "https://riffpad.ai",
    siteName: "Riffpad",
    locale: "zh_CN",
    type: "website",
    images: [
      {
        url: "/og.png",
        width: 1200,
        height: 630,
        alt: "Riffpad - AI 时代的代码灵感草稿本",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "Riffpad - AI 时代的代码灵感草稿本",
    description:
      "随时随地捕捉代码灵感，秒级启动隔离沙箱运行原型，一键桥接到 Cursor / Claude Code。",
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
    <html lang="zh-CN">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        {children}
      </body>
    </html>
  );
}
