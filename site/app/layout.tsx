import type { Metadata } from "next";
import { Poppins, Roboto } from "next/font/google";
import "./globals.css";

const poppins = Poppins({
  subsets: ["latin"],
  weight: "600",
  variable: "--font-poppins",
  display: "swap",
});

const roboto = Roboto({
  subsets: ["latin"],
  weight: ["400", "500", "700"],
  variable: "--font-roboto",
  display: "swap",
});

export const metadata: Metadata = {
  title: "RankOS - Print Farm Management & Order Fulfillment",
  description:
    "The all-in-one platform for managing your 3D print farm, tracking orders from Etsy & Squarespace, and running your maker business.",
  openGraph: {
    title: "RankOS - Print Farm Management & Order Fulfillment",
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
    <html lang="en" className={`${poppins.variable} ${roboto.variable}`}>
      <body className="bg-canvas font-body text-body antialiased">
        {children}
      </body>
    </html>
  );
}
