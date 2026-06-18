import { AdminLayout } from "@/components/shell/AdminLayout";
import { EmptyState } from "@/components/ui/EmptyState";

export default function AdminStatsPage() {
  return (
    <AdminLayout>
      <div className="grid gap-4 p-6 sm:grid-cols-2">
        <div className="rounded-lg border border-slate-200 bg-white p-6 shadow-sm">
          <p className="text-sm text-slate-600">Total users</p>
          <p className="mt-2 text-3xl font-semibold text-slate-900">—</p>
        </div>
        <div className="rounded-lg border border-slate-200 bg-white p-6 shadow-sm">
          <p className="text-sm text-slate-600">Total boards</p>
          <p className="mt-2 text-3xl font-semibold text-slate-900">—</p>
        </div>
      </div>
      <EmptyState
        title="Stats placeholder"
        description="Live admin statistics connect when the admin API lands in Part 8."
      />
    </AdminLayout>
  );
}
