"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const NAV = [
  { href: "/", label: "Cari", icon: "⌕" },
  { href: "/riwayat", label: "Riwayat", icon: "◷" },
  { href: "/pengaturan", label: "AI", icon: "✦" },
];

export function BottomNav() {
  const path = usePathname();
  return (
    <nav className="bottom-nav">
      {NAV.map((item) => {
        const active = path === item.href || (item.href !== "/" && path.startsWith(item.href));
        return (
          <Link key={item.href} href={item.href} aria-current={active ? "page" : undefined} className={`nav-item ${active ? "active" : ""}`} style={{ justifyContent: "center", fontSize: 12 }}>
            <span>{item.icon}</span>
            {item.label}
          </Link>
        );
      })}
    </nav>
  );
}
