import type { Metadata } from "next";
import "@fontsource/dm-sans/400.css";
import "@fontsource/dm-sans/500.css";
import "@fontsource/dm-sans/600.css";
import "@fontsource-variable/inter";
import "@fontsource/libre-baskerville/700.css";
import "@fontsource-variable/fraunces";
import "@fontsource-variable/manrope";
import "@fontsource-variable/source-sans-3";
import "@fontsource/space-mono/400.css";
import { connection } from "next/server";

import { AppProviders } from "@/app/providers";

import "./globals.css";

export const metadata: Metadata = {
  title: {
    default: "Monierave",
    template: "%s | Monierave",
  },
  description: "A calm, precise digital banking experience.",
};

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  // The request-specific CSP nonce can only be attached during dynamic rendering.
  await connection();

  return (
    <html lang="en">
      <head />
      <body>
        <AppProviders>{children}</AppProviders>
      </body>
    </html>
  );
}
