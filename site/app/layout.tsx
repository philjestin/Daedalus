import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Daedalus - Print Farm Management & Order Fulfillment",
  description:
    "The all-in-one platform for managing your 3D print farm, tracking orders from Etsy & Squarespace, and running your maker business.",
  openGraph: {
    title: "Daedalus - Print Farm Management & Order Fulfillment",
    description:
      "The all-in-one platform for managing your 3D print farm, tracking orders from Etsy & Squarespace, and running your maker business.",
    type: "website",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <head>
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link
          rel="preconnect"
          href="https://fonts.gstatic.com"
          crossOrigin="anonymous"
        />
        <link
          href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600&family=Space+Grotesk:wght@400;500;600;700&display=swap"
          rel="stylesheet"
        />
      </head>
      <body className="bg-surface-950 text-surface-100 antialiased">
        {children}
      </body>
    </html>
  );
}
