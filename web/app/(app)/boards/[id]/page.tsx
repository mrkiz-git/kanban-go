import { EmptyState } from "@/components/ui/EmptyState";

export function generateStaticParams() {
  return [{ id: "demo" }];
}

type BoardPageProps = {
  params: Promise<{ id: string }>;
};

export default async function BoardPage({ params }: BoardPageProps) {
  const { id } = await params;

  return (
    <div className="flex flex-1 flex-col">
      <header className="hidden border-b border-slate-200 bg-white px-6 py-4 lg:block">
        <h1 className="text-xl font-semibold text-slate-900">Board {id}</h1>
      </header>
      <EmptyState
        title="Board shell ready"
        description="Columns, cards, and drag-and-drop arrive in Part 5. This page confirms routing and the app shell layout."
      />
    </div>
  );
}
