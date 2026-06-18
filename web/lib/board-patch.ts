import type { Column } from "@/lib/boards";
import type { JsonPatchOp } from "@/lib/boards";

function columnIndex(columns: Column[], columnId: string) {
  return columns.findIndex((col) => col.id === columnId);
}

export function moveCardPatch(
  columns: Column[],
  sourceColumnId: string,
  sourceIndex: number,
  destColumnId: string,
  destIndex: number,
): JsonPatchOp[] {
  const srcIdx = columnIndex(columns, sourceColumnId);
  const dstIdx = columnIndex(columns, destColumnId);
  if (srcIdx < 0 || dstIdx < 0) {
    throw new Error("Column not found");
  }
  return [
    {
      op: "move",
      from: `/columns/${srcIdx}/cards/${sourceIndex}`,
      path: `/columns/${dstIdx}/cards/${destIndex}`,
    },
  ];
}

export function addCardPatch(columnIndex: number, title: string): JsonPatchOp[] {
  return [
    {
      op: "add",
      path: `/columns/${columnIndex}/cards/-`,
      value: { title, position: 0 },
    },
  ];
}

export function addColumnPatch(title: string): JsonPatchOp[] {
  return [
    {
      op: "add",
      path: "/columns/-",
      value: { title, position: 0, cards: [] },
    },
  ];
}

export function replaceBoardNamePatch(name: string): JsonPatchOp[] {
  return [{ op: "replace", path: "/name", value: name }];
}

export function replaceCardFieldPatch(
  columns: Column[],
  columnId: string,
  cardId: string,
  field: "title" | "description",
  value: string,
): JsonPatchOp[] {
  const colIdx = columnIndex(columns, columnId);
  if (colIdx < 0) {
    throw new Error("Column not found");
  }
  const cardIdx = columns[colIdx].cards.findIndex((card) => card.id === cardId);
  if (cardIdx < 0) {
    throw new Error("Card not found");
  }
  return [{ op: "replace", path: `/columns/${colIdx}/cards/${cardIdx}/${field}`, value }];
}
