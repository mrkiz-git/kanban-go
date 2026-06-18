import { Button } from "@/components/ui/Button";
import { EmptyState } from "@/components/ui/EmptyState";

export default function BoardListPage() {
  return (
    <EmptyState
      title="No boards yet"
      description="Create your first Kanban board to start organizing work. Board management connects in Part 5."
      action={<Button variant="secondary">+ New Board</Button>}
    />
  );
}
