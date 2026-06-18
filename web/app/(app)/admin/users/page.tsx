import { AdminLayout } from "@/components/shell/AdminLayout";
import { EmptyState } from "@/components/ui/EmptyState";

export default function AdminUsersPage() {
  return (
    <AdminLayout>
      <EmptyState
        title="User management"
        description="The admin users table and edit modals will be wired up in Part 8."
      />
    </AdminLayout>
  );
}
