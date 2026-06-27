"use client";

import { useDroppable } from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { SortableCard } from "@/components/board/SortableCard";
import type { Card, Column } from "@/lib/boards";

type BoardColumnProps = {
  column: Column;
  readOnly: boolean;
  addingCardColumnId: string | null;
  onSelectCard: (columnId: string, card: Card) => void;
  onStartAddCard: (columnId: string) => void;
  addCardForm: React.ReactNode;
  addCardButton: React.ReactNode;
};

export function BoardColumn({
  column,
  readOnly,
  addingCardColumnId,
  onSelectCard,
  onStartAddCard,
  addCardForm,
  addCardButton,
}: BoardColumnProps) {
  const { setNodeRef, isOver } = useDroppable({
    id: column.id,
    disabled: readOnly,
  });
  const cardIds = column.cards.map((card) => card.id);

  return (
    <div className="flex w-72 shrink-0 flex-col rounded-lg bg-slate-100">
      <div className="flex items-center justify-between px-3 py-3">
        <h2 className="text-lg font-semibold text-slate-900">{column.title}</h2>
        <span className="text-xs text-slate-600">{column.cards.length}</span>
      </div>

      <div
        ref={setNodeRef}
        className={`flex min-h-32 flex-1 flex-col gap-2 px-2 pb-2 ${
          isOver ? "bg-blue-50/60" : ""
        }`}
      >
        {column.cards.length === 0 && !readOnly && addingCardColumnId !== column.id ? (
          <button
            type="button"
            className="flex flex-1 items-center justify-center rounded-lg border border-dashed border-slate-300 bg-white/60 px-3 py-6 text-sm text-slate-600 hover:border-blue-400 hover:bg-blue-50/50 hover:text-blue-700"
            onClick={() => onStartAddCard(column.id)}
          >
            + Add first card
          </button>
        ) : null}
        <SortableContext items={cardIds} strategy={verticalListSortingStrategy}>
          {column.cards.map((card) => (
            <SortableCard
              key={card.id}
              card={card}
              readOnly={readOnly}
              onSelect={() => onSelectCard(column.id, card)}
            />
          ))}
        </SortableContext>
      </div>

      {!readOnly ? (addingCardColumnId === column.id ? addCardForm : addCardButton) : null}
    </div>
  );
}
