"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";

type AdminLayoutProps = {
  children: ReactNode;
};

const tabs = [
  { href: "/admin/users/", label: "Users" },
  { href: "/admin/stats/", label: "Stats" },
];

export function AdminLayout({ children }: AdminLayoutProps) {
  const pathname = usePathname();

  return (
    <div className="flex flex-1 flex-col">
      <div className="border-b border-slate-200 bg-white px-6 py-4">
        <h1 className="text-xl font-semibold text-slate-900">Admin Panel</h1>
        <nav className="mt-4 flex gap-2">
          {tabs.map((tab) => {
            const active = pathname === tab.href;
            return (
              <Link
                key={tab.href}
                href={tab.href}
                className={`rounded px-3 py-2 text-sm font-medium ${
                  active
                    ? "bg-blue-50 text-blue-700"
                    : "text-slate-700 hover:bg-slate-100"
                }`}
              >
                {tab.label}
              </Link>
            );
          })}
        </nav>
      </div>
      <div className="flex-1 overflow-auto">{children}</div>
    </div>
  );
}
