"use client";

import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import type { Card } from "@/lib/boards";

type SortableCardProps = {
  card: Card;
  readOnly: boolean;
  onSelect: () => void;
};

export function SortableCard({ card, readOnly, onSelect }: SortableCardProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: card.id,
    disabled: readOnly,
    data: { type: "card" },
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  return (
    <button
      type="button"
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      className={`rounded-lg border border-slate-200 bg-white p-3 text-left shadow-sm transition-shadow ${
        isDragging ? "opacity-40 shadow-lg" : ""
      }`}
      onClick={onSelect}
    >
      <p className="text-sm font-medium text-slate-900">{card.title}</p>
      {card.description ? (
        <p className="mt-1 line-clamp-2 text-xs text-slate-600">{card.description}</p>
      ) : null}
    </button>
  );
}

export function CardPreview({ card }: { card: Card }) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-3 text-left shadow-lg">
      <p className="text-sm font-medium text-slate-900">{card.title}</p>
      {card.description ? (
        <p className="mt-1 line-clamp-2 text-xs text-slate-600">{card.description}</p>
      ) : null}
    </div>
  );
}
