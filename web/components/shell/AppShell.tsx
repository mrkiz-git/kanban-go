"use client";

import { useState, type ReactNode } from "react";
import { AiSidebar } from "./AiSidebar";
import { Sidebar } from "./Sidebar";

type AppShellProps = {
  children: ReactNode;
  title?: string;
};

export function AppShell({ children, title }: AppShellProps) {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  return (
    <div className="flex min-h-screen bg-slate-50">
      <Sidebar open={sidebarOpen} onClose={() => setSidebarOpen(false)} />

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex items-center gap-3 border-b border-slate-200 bg-white px-4 py-3 lg:hidden">
          <button
            type="button"
            aria-label="Open navigation"
            className="rounded px-2 py-1 text-lg text-slate-700 hover:bg-slate-100"
            onClick={() => setSidebarOpen(true)}
          >
            ≡
          </button>
          {title ? <h1 className="truncate text-lg font-semibold text-slate-900">{title}</h1> : null}
        </header>

        <div className="flex min-h-0 flex-1">
          <main className="flex min-w-0 flex-1 flex-col overflow-hidden">{children}</main>
          <AiSidebar />
        </div>
      </div>
    </div>
  );
}
