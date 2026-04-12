import type { Metadata } from "next";
import { Plus_Jakarta_Sans } from "next/font/google";
import { Geist_Mono } from "next/font/google";
import "./globals.css";
import { Navbar } from "@/components/navbar";
import { Providers } from "@/components/providers";

const jakarta = Plus_Jakarta_Sans({
  variable: "--font-sans",
  subsets: ["latin"],
  weight: ["400", "500", "600", "700", "800"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Auction — Real-time Bidding Platform",
  description: "Discover, bid, and win unique items in real-time auctions",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`${jakarta.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col">
        <Providers>
          <Navbar />
          <main className="flex-1 w-full max-w-[1200px] mx-auto px-5 sm:px-8 py-8">{children}</main>
        </Providers>
      </body>
    </html>
  );
}
