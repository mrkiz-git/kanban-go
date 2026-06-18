"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "@/lib/auth";

type SidebarProps = {
  open: boolean;
  onClose: () => void;
};

function initials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length === 0) {
    return "?";
  }
  if (parts.length === 1) {
    return parts[0].slice(0, 2).toUpperCase();
  }
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

export function Sidebar({ open, onClose }: SidebarProps) {
  const pathname = usePathname();
  const { user, logout } = useAuth();

  return (
    <>
      {open ? (
        <button
          type="button"
          aria-label="Close navigation"
          className="fixed inset-0 z-40 bg-black/40 lg:hidden"
          onClick={onClose}
        />
      ) : null}

      <aside
        className={`fixed inset-y-0 left-0 z-50 flex w-60 flex-col bg-slate-100 transition-transform lg:static lg:translate-x-0 ${
          open ? "translate-x-0" : "-translate-x-full"
        }`}
      >
        <div className="border-b border-slate-200 px-4 py-5">
          <Link href="/boards/" className="text-xl font-bold text-slate-900">
            Kanba
          </Link>
        </div>

        <div className="flex-1 overflow-y-auto px-3 py-4">
          <p className="px-2 text-xs font-semibold uppercase tracking-wide text-slate-600">
            My Boards
          </p>
          <p className="mt-2 px-2 text-sm text-slate-600">No boards yet.</p>

          <button
            type="button"
            className="mt-4 w-full rounded px-2 py-2 text-left text-sm text-slate-700 hover:bg-slate-200"
          >
            + New Board
          </button>
        </div>

        <div className="border-t border-slate-200 px-4 py-4">
          {user?.role === "admin" ? (
            <Link
              href="/admin/users/"
              className="mb-3 block text-sm text-slate-700 hover:text-slate-900"
            >
              Admin
            </Link>
          ) : null}
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-full bg-blue-600 text-sm font-medium text-white">
              {user ? initials(user.name) : "?"}
            </div>
            <div>
              <p className="text-sm font-medium text-slate-900">{user?.name ?? "Guest"}</p>
              <button
                type="button"
                className="text-xs text-slate-600 hover:text-slate-900"
                onClick={() => logout()}
              >
                Sign out
              </button>
            </div>
          </div>
        </div>
      </aside>
    </>
  );
}
