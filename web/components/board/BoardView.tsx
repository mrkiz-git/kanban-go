"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  DndContext,
  DragOverlay,
  KeyboardSensor,
  PointerSensor,
  closestCorners,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragOverEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import {
  arrayMove,
  sortableKeyboardCoordinates,
} from "@dnd-kit/sortable";
import { useRouter } from "next/navigation";
import { BoardColumn } from "@/components/board/BoardColumn";
import { CardModal, findCardColumn } from "@/components/board/CardModal";
import { CardPreview } from "@/components/board/SortableCard";
import { Button } from "@/components/ui/Button";
import { EmptyState } from "@/components/ui/EmptyState";
import { addCardPatch, addColumnPatch, moveCardPatch, replaceBoardNamePatch } from "@/lib/board-patch";
import { APIError } from "@/lib/api";
import {
  deleteBoard,
  getBoard,
  isReadOnly,
  listBoards,
  patchBoard,
  type Board,
  type Card,
  type Column,
  type JsonPatchOp,
} from "@/lib/boards";

function findColumnId(columns: Column[], id: string): string | null {
  if (columns.some((col) => col.id === id)) {
    return id;
  }
  for (const col of columns) {
    if (col.cards.some((card) => card.id === id)) {
      return col.id;
    }
  }
  return null;
}
import { useBoards } from "@/lib/boards-context";

type BoardViewProps = {
  boardId: string;
};

export function BoardView({ boardId }: BoardViewProps) {
  const router = useRouter();
  const { boards, refreshBoards } = useBoards();
  const [board, setBoard] = useState<Board | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [selectedCard, setSelectedCard] = useState<{ columnId: string; card: Card } | null>(
    null,
  );
  const [editingTitle, setEditingTitle] = useState(false);
  const [titleDraft, setTitleDraft] = useState("");
  const [addingCardColumnId, setAddingCardColumnId] = useState<string | null>(null);
  const [newCardTitle, setNewCardTitle] = useState("");
  const [addingColumn, setAddingColumn] = useState(false);
  const [newColumnTitle, setNewColumnTitle] = useState("");
  const [showCardPicker, setShowCardPicker] = useState(false);
  const [activeCard, setActiveCard] = useState<Card | null>(null);
  const addColumnRef = useRef<HTMLDivElement>(null);
  const boardScrollRef = useRef<HTMLDivElement>(null);
  const dragStartBoard = useRef<Board | null>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const permission = boards.find((item) => item.id === boardId)?.permission;
  const readOnly = isReadOnly(permission);

  const loadBoard = useCallback(async () => {
    setError("");
    try {
      const data = await getBoard(boardId);
      setBoard(data);
      setTitleDraft(data.name);
    } catch (err) {
      if (err instanceof APIError && err.status === 404) {
        setError("Board not found");
      } else {
        setError(err instanceof Error ? err.message : "Failed to load board");
      }
      setBoard(null);
    } finally {
      setLoading(false);
    }
  }, [boardId]);

  useEffect(() => {
    setLoading(true);
    void loadBoard();
    void refreshBoards();
  }, [loadBoard, refreshBoards]);

  useEffect(() => {
    if (!showCardPicker) {
      return;
    }
    function onPointerDown(event: MouseEvent) {
      const target = event.target as HTMLElement;
      if (!target.closest("[data-card-picker]")) {
        setShowCardPicker(false);
      }
    }
    document.addEventListener("mousedown", onPointerDown);
    return () => document.removeEventListener("mousedown", onPointerDown);
  }, [showCardPicker]);

  function startAddCard(columnId: string) {
    setShowCardPicker(false);
    setAddingCardColumnId(columnId);
    setNewCardTitle("");
  }

  function startAddColumn() {
    setAddingColumn(true);
    setNewColumnTitle("");
    requestAnimationFrame(() => {
      addColumnRef.current?.scrollIntoView({ behavior: "smooth", inline: "end" });
    });
  }

  async function handleBoardNotFound() {
    const latest = await listBoards();
    if (latest.boards.some((item) => item.id === boardId)) {
      setLoading(true);
      await loadBoard();
      return;
    }
    await refreshBoards();
    router.replace("/boards/");
  }

  async function applyPatch(patch: JsonPatchOp[], optimistic: Board) {
    if (!board) {
      return;
    }
    setBoard(optimistic);
    try {
      const updated = await patchBoard(board.id, board.version, patch);
      setBoard(updated);
      await refreshBoards();
    } catch (err) {
      setBoard(board);
      if (err instanceof APIError && err.status === 409) {
        await loadBoard();
        setError("Board changed elsewhere. Refreshed to the latest version.");
      } else {
        setError(err instanceof Error ? err.message : "Update failed");
      }
    }
  }

  function handleDragStart(event: DragStartEvent) {
    if (!board) {
      return;
    }
    dragStartBoard.current = board;
    const card = board.columns
      .flatMap((col) => col.cards)
      .find((item) => item.id === event.active.id);
    setActiveCard(card ?? null);
  }

  function handleDragOver(event: DragOverEvent) {
    const { active, over } = event;
    if (!board || readOnly || !over) {
      return;
    }

    const activeId = String(active.id);
    const overId = String(over.id);
    const activeColumnId = findColumnId(board.columns, activeId);
    const overColumnId = findColumnId(board.columns, overId);
    if (!activeColumnId || !overColumnId) {
      return;
    }

    if (activeColumnId === overColumnId) {
      const column = board.columns.find((col) => col.id === activeColumnId);
      if (!column) {
        return;
      }
      const activeIndex = column.cards.findIndex((card) => card.id === activeId);
      const overIndex = column.cards.findIndex((card) => card.id === overId);
      if (activeIndex < 0 || overIndex < 0 || activeIndex === overIndex) {
        return;
      }

      setBoard((prev) => {
        if (!prev) {
          return prev;
        }
        const next = structuredClone(prev);
        const targetCol = next.columns.find((col) => col.id === activeColumnId);
        if (!targetCol) {
          return prev;
        }
        targetCol.cards = arrayMove(targetCol.cards, activeIndex, overIndex);
        targetCol.cards.forEach((card, index) => {
          card.position = index;
        });
        return next;
      });
      return;
    }

    setBoard((prev) => {
      if (!prev) {
        return prev;
      }
      const next = structuredClone(prev);
      const sourceCol = next.columns.find((col) => col.id === activeColumnId);
      const destCol = next.columns.find((col) => col.id === overColumnId);
      if (!sourceCol || !destCol) {
        return prev;
      }

      const activeIndex = sourceCol.cards.findIndex((card) => card.id === activeId);
      if (activeIndex < 0) {
        return prev;
      }

      let overIndex = destCol.cards.findIndex((card) => card.id === overId);
      if (overIndex < 0) {
        overIndex = destCol.cards.length;
      }

      const [moved] = sourceCol.cards.splice(activeIndex, 1);
      destCol.cards.splice(overIndex, 0, moved);
      sourceCol.cards.forEach((card, index) => {
        card.position = index;
      });
      destCol.cards.forEach((card, index) => {
        card.position = index;
      });
      return next;
    });
  }

  function handleDragEnd(event: DragEndEvent) {
    setActiveCard(null);
    const start = dragStartBoard.current;
    dragStartBoard.current = null;

    if (!board || readOnly || !start) {
      return;
    }

    const { active, over } = event;
    if (!over) {
      setBoard(start);
      return;
    }

    const activeId = String(active.id);
    const startCol = start.columns.find((col) => col.cards.some((card) => card.id === activeId));
    const endCol = board.columns.find((col) => col.cards.some((card) => card.id === activeId));
    if (!startCol || !endCol) {
      setBoard(start);
      return;
    }

    const startIndex = startCol.cards.findIndex((card) => card.id === activeId);
    const endIndex = endCol.cards.findIndex((card) => card.id === activeId);
    if (startCol.id === endCol.id && startIndex === endIndex) {
      return;
    }

    const patch = moveCardPatch(start.columns, startCol.id, startIndex, endCol.id, endIndex);
    void applyPatch(patch, board);
  }

  function handleDragCancel() {
    if (dragStartBoard.current) {
      setBoard(dragStartBoard.current);
    }
    dragStartBoard.current = null;
    setActiveCard(null);
  }

  async function handleAddCard(columnId: string, title = "New card") {
    if (!board || readOnly) {
      return;
    }
    const trimmed = title.trim();
    if (!trimmed) {
      return;
    }
    const colIdx = board.columns.findIndex((col) => col.id === columnId);
    if (colIdx < 0) {
      return;
    }
    const patch = addCardPatch(colIdx, trimmed);
    try {
      const updated = await patchBoard(board.id, board.version, patch);
      setBoard(updated);
      setAddingCardColumnId(null);
      setNewCardTitle("");
    } catch (err) {
      if (err instanceof APIError && err.status === 409) {
        await loadBoard();
      }
      setError(err instanceof Error ? err.message : "Failed to add card");
    }
  }

  async function handleAddColumn(title = "New column") {
    if (!board || readOnly) {
      return;
    }
    const trimmed = title.trim();
    if (!trimmed) {
      return;
    }
    if (board.columns.length >= 20) {
      setError("A board can have at most 20 columns");
      return;
    }
    const patch = addColumnPatch(trimmed);
    try {
      const updated = await patchBoard(board.id, board.version, patch);
      setBoard(updated);
      setAddingColumn(false);
      setNewColumnTitle("");
    } catch (err) {
      if (err instanceof APIError && err.status === 409) {
        await loadBoard();
      }
      setError(err instanceof Error ? err.message : "Failed to add column");
    }
  }

  async function handleRenameBoard() {
    if (!board || readOnly || titleDraft.trim() === board.name) {
      setEditingTitle(false);
      return;
    }
    try {
      const updated = await patchBoard(
        board.id,
        board.version,
        replaceBoardNamePatch(titleDraft.trim()),
      );
      setBoard(updated);
      await refreshBoards();
      setEditingTitle(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to rename board");
    }
  }

  async function handleDeleteBoard() {
    if (!board || permission !== "owner") {
      return;
    }
    if (!window.confirm(`Delete board "${board.name}"?`)) {
      return;
    }
    try {
      await deleteBoard(board.id);
      await refreshBoards();
      router.replace("/boards/");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete board");
    }
  }

  if (loading) {
    return (
      <div className="flex flex-1 items-center justify-center text-sm text-slate-600">
        Loading board…
      </div>
    );
  }

  if (error && !board) {
    return (
      <EmptyState
        title="Unable to load board"
        description={error}
        action={
          <div className="flex flex-wrap justify-center gap-2">
            <Button variant="primary" onClick={() => void handleBoardNotFound()}>
              Try again
            </Button>
            <Button variant="secondary" onClick={() => router.push("/boards/")}>
              Back to boards
            </Button>
          </div>
        }
      />
    );
  }

  if (!board) {
    return null;
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <header className="border-b border-slate-200 bg-white px-4 py-4 lg:px-6">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="min-w-0 flex-1">
            {editingTitle && !readOnly ? (
              <input
                className="w-full max-w-md rounded border border-slate-200 px-2 py-1 text-xl font-semibold text-slate-900 outline-none focus:border-blue-600"
                value={titleDraft}
                onChange={(e) => setTitleDraft(e.target.value)}
                onBlur={() => void handleRenameBoard()}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    void handleRenameBoard();
                  }
                  if (e.key === "Escape") {
                    setTitleDraft(board.name);
                    setEditingTitle(false);
                  }
                }}
                autoFocus
              />
            ) : (
              <button
                type="button"
                className="truncate text-left text-xl font-semibold text-slate-900 hover:text-blue-700"
                onClick={() => !readOnly && setEditingTitle(true)}
                disabled={readOnly}
              >
                {board.name}
              </button>
            )}
            {readOnly ? (
              <p className="mt-1 text-xs text-slate-500">Read-only access</p>
            ) : null}
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {!readOnly ? (
              <>
                <div className="relative" data-card-picker>
                  <Button type="button" onClick={() => setShowCardPicker((open) => !open)}>
                    + Add card
                  </Button>
                  {showCardPicker ? (
                    <div
                      className="absolute right-0 z-20 mt-2 min-w-44 rounded-lg border border-slate-200 bg-white py-1 shadow-md"
                      data-card-picker
                    >
                      <p className="px-3 py-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
                        Choose column
                      </p>
                      {board.columns.map((column) => (
                        <button
                          key={column.id}
                          type="button"
                          className="block w-full px-3 py-2 text-left text-sm text-slate-700 hover:bg-slate-100"
                          onClick={() => startAddCard(column.id)}
                        >
                          {column.title}
                        </button>
                      ))}
                    </div>
                  ) : null}
                </div>
                <Button type="button" variant="secondary" onClick={startAddColumn}>
                  + Add column
                </Button>
              </>
            ) : null}
            {permission === "owner" ? (
              <Button variant="ghost" onClick={() => void handleDeleteBoard()}>
                Delete board
              </Button>
            ) : null}
          </div>
        </div>
      </header>

      {error ? (
        <div className="border-b border-red-200 bg-red-50 px-4 py-2 text-sm text-red-600">
          {error}
        </div>
      ) : null}

      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        onDragStart={handleDragStart}
        onDragOver={handleDragOver}
        onDragEnd={handleDragEnd}
        onDragCancel={handleDragCancel}
      >
        <div ref={boardScrollRef} className="flex min-h-0 flex-1 gap-4 overflow-x-auto p-4">
          {board.columns.map((column) => (
            <BoardColumn
              key={column.id}
              column={column}
              readOnly={readOnly}
              addingCardColumnId={addingCardColumnId}
              onSelectCard={(columnId, card) => setSelectedCard({ columnId, card })}
              onStartAddCard={startAddCard}
              addCardForm={
                <form
                  className="mx-2 mb-3 space-y-2"
                  onSubmit={(event) => {
                    event.preventDefault();
                    void handleAddCard(column.id, newCardTitle);
                  }}
                >
                  <input
                    className="w-full rounded border border-slate-200 px-2 py-1.5 text-sm outline-none focus:border-blue-600"
                    placeholder="Card title"
                    value={newCardTitle}
                    onChange={(e) => setNewCardTitle(e.target.value)}
                    maxLength={200}
                    autoFocus
                  />
                  <div className="flex gap-2">
                    <Button
                      type="submit"
                      className="px-3 py-1 text-xs"
                      disabled={!newCardTitle.trim()}
                    >
                      Add card
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      className="px-3 py-1 text-xs"
                      onClick={() => {
                        setAddingCardColumnId(null);
                        setNewCardTitle("");
                      }}
                    >
                      Cancel
                    </Button>
                  </div>
                </form>
              }
              addCardButton={
                <button
                  type="button"
                  className="mx-2 mb-3 rounded border border-slate-200 bg-white px-2 py-2 text-left text-sm font-medium text-blue-700 hover:bg-blue-50"
                  onClick={() => startAddCard(column.id)}
                >
                  + Add a card
                </button>
              }
            />
          ))}

          {!readOnly ? (
            <div ref={addColumnRef}>
              {addingColumn ? (
              <form
                className="flex w-72 shrink-0 flex-col rounded-lg border border-dashed border-slate-300 bg-white p-3"
                onSubmit={(event) => {
                  event.preventDefault();
                  void handleAddColumn(newColumnTitle);
                }}
              >
                <input
                  className="rounded border border-slate-200 px-2 py-1.5 text-sm outline-none focus:border-blue-600"
                  placeholder="Column title"
                  value={newColumnTitle}
                  onChange={(e) => setNewColumnTitle(e.target.value)}
                  maxLength={100}
                  autoFocus
                />
                <div className="mt-2 flex gap-2">
                  <Button type="submit" className="px-3 py-1 text-xs" disabled={!newColumnTitle.trim()}>
                    Add column
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    className="px-3 py-1 text-xs"
                    onClick={() => {
                      setAddingColumn(false);
                      setNewColumnTitle("");
                    }}
                  >
                    Cancel
                  </Button>
                </div>
              </form>
            ) : (
              <button
                type="button"
                className="flex h-fit w-72 shrink-0 flex-col items-center justify-center rounded-lg border-2 border-dashed border-blue-200 bg-blue-50/40 px-4 py-10 text-sm font-medium text-blue-700 hover:bg-blue-50"
                onClick={startAddColumn}
              >
                + Add column
              </button>
            )}
            </div>
          ) : null}
        </div>
        <DragOverlay>{activeCard ? <CardPreview card={activeCard} /> : null}</DragOverlay>
      </DndContext>

      {selectedCard ? (
        <CardModal
          board={board}
          columnId={selectedCard.columnId}
          card={selectedCard.card}
          readOnly={readOnly}
          onClose={() => setSelectedCard(null)}
          onSaved={(updated) => {
            setBoard(updated);
            const col = findCardColumn(updated.columns, selectedCard.card.id);
            const card = col?.cards.find((item) => item.id === selectedCard.card.id);
            if (card && col) {
              setSelectedCard({ columnId: col.id, card });
            }
          }}
        />
      ) : null}
    </div>
  );
}
