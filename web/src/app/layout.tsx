import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "BPMN Impact Mapper",
  description: "Evidence-led BPMN change impact analysis",
};

const THEME_INIT = `(function(){try{var t=localStorage.getItem("bpmn-impact-theme");if(t==="dark"||t==="light"){document.documentElement.dataset.theme=t;}}catch(e){}})();`;

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: THEME_INIT }} />
      </head>
      <body>{children}</body>
    </html>
  );
}
