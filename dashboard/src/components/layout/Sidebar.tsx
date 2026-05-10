"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ThemeToggle } from "@/components/theme/theme-toggle";

const NAV = [
  { href: "/", label: "Discover", helper: "Search products", icon: "⌕" },
  { href: "/riwayat", label: "History", helper: "Past scans", icon: "◷" },
  { href: "/pengaturan", label: "AI Settings", helper: "Provider & model", icon: "✦" },
];

export function Sidebar() {
  const path = usePathname();
  return (
    <aside className="sidebar" aria-label="Primary navigation">
      <Link href="/" className="sidebar-brand">
        <span className="brand-mark">P</span>
        <span style={{ minWidth: 0 }}>
          <span style={{ display: "block", fontSize: 21, fontWeight: 950, letterSpacing: "-0.06em", lineHeight: 1 }}>PriceScope</span>
          <span className="text-faint" style={{ display: "block", fontSize: 12, marginTop: 4 }}>Blue market intelligence</span>
        </span>
      </Link>

      <nav style={{ display: "grid", gap: 7 }}>
        <p className="sidebar-section-label">Workspace</p>
        {NAV.map((item) => {
          const active = path === item.href || (item.href !== "/" && path.startsWith(item.href));
          return (
            <Link key={item.href} href={item.href} aria-current={active ? "page" : undefined} className={`nav-item ${active ? "active" : ""}`}>
              <span className="nav-icon" aria-hidden="true">{item.icon}</span>
              <span className="nav-copy">
                <span>{item.label}</span>
                <small>{item.helper}</small>
              </span>
            </Link>
          );
        })}
      </nav>

      <div className="sidebar-card" style={{ display: "grid", gap: 11 }}>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12 }}>
          <span className="text-faint" style={{ fontSize: 11, fontWeight: 850, letterSpacing: ".12em", textTransform: "uppercase" }}>Signal</span>
          <span className="marketplace-badge mp-default">Live</span>
        </div>
        <p style={{ margin: 0, fontSize: 13.5, lineHeight: 1.5, fontWeight: 740 }}>AI insights, grouping, and price radar for fast marketplace decisions.</p>
        <div className="sidebar-meter" aria-hidden="true"><span /></div>
      </div>

      <div style={{ flex: 1 }} />

      <div className="sidebar-card" style={{ display: "grid", gap: 12 }}>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 10 }}>
          <div>
            <p style={{ margin: 0, fontSize: 13, fontWeight: 850 }}>Interface tone</p>
            <p className="text-faint" style={{ margin: "2px 0 0", fontSize: 11 }}>Blue creative mode</p>
          </div>
          <ThemeToggle />
        </div>
      </div>
    </aside>
  );
}
