import type { Metadata } from "next";
import "@fontsource-variable/fraunces";
import "@fontsource-variable/source-sans-3";

import { AppProviders } from "@/app/providers";

import "./globals.css";

export const metadata: Metadata = {
  title: {
    default: "Monierave",
    template: "%s | Monierave",
  },
  description: "A calm, precise digital banking experience.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body>
        <AppProviders>{children}</AppProviders>
      </body>
    </html>
  );
}
