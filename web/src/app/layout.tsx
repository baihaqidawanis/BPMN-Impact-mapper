import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "BPMN Impact Mapper",
  description: "Evidence-led BPMN change impact analysis",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
