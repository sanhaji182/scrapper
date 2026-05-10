import type { Metadata } from "next";
import { ThemeProvider } from "@/components/theme/theme-provider";
import { Sidebar } from "@/components/layout/Sidebar";
import { BottomNav } from "@/components/layout/BottomNav";
import "./globals.css";

export const metadata: Metadata = {
  title: "PriceScope — Bandingkan Harga Marketplace",
  description: "Cari dan bandingkan harga produk dari Tokopedia dan marketplace lainnya.",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="id" data-scroll-behavior="smooth" suppressHydrationWarning>
      <body>
        <ThemeProvider>
          <div className="app-shell">
            <Sidebar />
            <main className="main-panel">{children}</main>
          </div>
          <BottomNav />
        </ThemeProvider>
      </body>
    </html>
  );
}
