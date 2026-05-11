"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ThemeToggle } from "@/components/theme/theme-toggle";

const NAV = [
  { href: "/", label: "Discover", helper: "Search", icon: "⌕" },
  { href: "/riwayat", label: "History", helper: "Runs", icon: "◷" },
  { href: "/pengaturan", label: "AI Settings", helper: "Config", icon: "✦" },
  { href: "/runs", label: "Live Signal", helper: "Queue", icon: "◇" },
];

export function Sidebar() {
  const path = usePathname();
  return (
    <aside className="sidebar ps-sidebar" aria-label="Primary navigation">
      <Link href="/" className="sidebar-brand ps-sidebar-brand">
        <span className="brand-mark"><span aria-hidden="true">P</span></span>
        <span className="ps-brand-copy">
          <span>PriceScope</span>
          <small>AI price intelligence</small>
        </span>
      </Link>

      <nav className="ps-sidebar-nav">
        {NAV.map((item) => {
          const active = path === item.href || (item.href !== "/" && path.startsWith(item.href));
          return (
            <Link key={item.href} href={item.href} aria-current={active ? "page" : undefined} className={`nav-item ps-nav-item ${active ? "active" : ""}`}>
              <span className="nav-icon" aria-hidden="true">{item.icon}</span>
              <span className="nav-copy"><span>{item.label}</span><small>{item.helper}</small></span>
            </Link>
          );
        })}
      </nav>

      <div className="ps-sidebar-signal">
        <span>Live Signal</span>
        <strong>Marketplace radar active</strong>
        <div className="sidebar-meter" aria-hidden="true"><span /></div>
      </div>

      <div style={{ flex: 1 }} />

      <div className="ps-sidebar-bottom">
        <ThemeToggle />
        <div className="ps-user-chip"><span>SH</span><div><strong>User</strong><small>Local workspace</small></div></div>
      </div>
    </aside>
  );
}
