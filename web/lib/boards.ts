import { api } from "@/lib/api";

export type BoardPermission = "owner" | "write" | "read";

export type BoardSummary = {
  id: string;
  name: string;
  permission: BoardPermission;
  version: number;
  updatedAt: string;
};

export type Attachment = {
  id: string;
  filename: string;
  mimeType: string;
  sizeBytes: number;
  url: string;
  createdAt: string;
};

export type Card = {
  id: string;
  title: string;
  description?: string;
  position: number;
  attachments?: Attachment[];
  updatedAt: string;
};

export type Column = {
  id: string;
  title: string;
  position: number;
  cards: Card[];
};

export type Board = {
  id: string;
  name: string;
  version: number;
  columns: Column[];
  updatedAt: string;
};

export type JsonPatchOp = Record<string, unknown>;

export function listBoards() {
  return api<{ boards: BoardSummary[] }>("/boards");
}

export function createBoard(name: string) {
  return api<Board>("/boards", { method: "POST", body: { name } });
}

export function getBoard(id: string) {
  return api<Board>(`/boards/${id}`);
}

export function updateBoard(id: string, version: number, board: Board) {
  const { id: _id, version: _version, updatedAt: _updatedAt, ...body } = board;
  return api<Board>(`/boards/${id}`, {
    method: "PUT",
    headers: { "If-Match": `"${version}"` },
    body,
  });
}

export function patchBoard(id: string, version: number, patch: JsonPatchOp[]) {
  return api<Board>(`/boards/${id}`, {
    method: "PATCH",
    headers: { "If-Match": `"${version}"` },
    body: patch,
  });
}

export function deleteBoard(id: string) {
  return api<void>(`/boards/${id}`, { method: "DELETE" });
}

export function canWrite(permission: BoardPermission | undefined) {
  return permission !== "read";
}

export function isReadOnly(permission: BoardPermission | undefined) {
  return permission === "read";
}
