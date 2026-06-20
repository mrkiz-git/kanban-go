# UX/UI Review — Kanba Go
**Reviewer:** UX/UI Director  
**Date:** 2026-06-20  
**App version:** Part 5–6 build (board + kanban UI)  
**Method:** Source code audit (Next.js frontend) + live server inspection at `localhost:8080`

---

## Executive Summary

The app has a clean visual foundation — good Tailwind color tokens, sensible layout, and a working multi-board sidebar. However, there are critical interaction gaps that make the day-to-day Kanban workflow painful. The most serious is a **complete drag-and-drop regression** confirmed by the user. Six additional UX deficiencies follow in priority order.

---

## Issue 1 — Drag-and-Drop Is Broken (Confirmed Regression)

**Severity:** Critical — the core Kanban interaction is unusable.

**Root cause:** The project uses `@hello-pangea/dnd` v18.0.1 with React 19. The `@hello-pangea/dnd` library does not officially support React 19 and has known breakage with its concurrent rendering internals. The `next.config.ts` uses `output: "export"` (static export), which means the bundle is pre-compiled by Next.js 15 running React 19 — the DnD context collapses silently.

**Code location:** [`web/components/board/BoardView.tsx:375`](web/components/board/BoardView.tsx)  
**Relevant code:**  
```tsx
<DragDropContext onDragEnd={handleDragEnd}>
```

**Fix recommendation:**  
- Pin `react` and `react-dom` to `18.x` in `package.json`, OR  
- Replace `@hello-pangea/dnd` with `dnd-kit` (`@dnd-kit/core` + `@dnd-kit/sortable`), which has full React 19 support and a more modern API.

---

## Issue 2 — No Way to Delete or Move a Card

**Severity:** High — users are stuck with cards they can no longer drag (see Issue 1) and have no alternative escape route.

**What's missing:**  
- The `CardModal` ([`web/components/board/CardModal.tsx`](web/components/board/CardModal.tsx)) has **Save** and **Close** buttons only — no **Delete** and no **Move to column** selector.  
- Even when drag-and-drop is working, users expect to be able to move or remove a card from within the detail view. This is standard in Jira, Linear, Trello, and every major Kanban tool.

**Fix recommendation:**  
- Add a **"Move to…"** dropdown in the modal header to change the card's column.  
- Add a **Delete card** button (with confirmation) in the modal footer.  
- Reference: the `replaceCardFieldPatch` helper already exists; add a `moveCardToColumnPatch` and a `removeCardPatch` alongside it in `board-patch.ts`.

---

## Issue 3 — Columns Cannot Be Renamed or Deleted

**Severity:** High — column lifecycle management is entirely missing.

**What's missing:**  
The `BoardView` renders column titles as static `<h2>` text ([`web/components/board/BoardView.tsx:383`](web/components/board/BoardView.tsx)). There is no:
- Inline rename (click to edit, press Enter to save)
- Delete column action
- Column reordering (columns can't be dragged either — only cards have `<Draggable>` wrappers)

**Fix recommendation:**  
- Make `<h2>` an inline-editable element (same pattern as board title rename already implemented at line 295–321).
- Add a `⋯` overflow menu per column with "Rename" and "Delete" options.
- Wrap each column `<div>` in its own `<Draggable>` to support column reordering. The `@hello-pangea/dnd` API supports nested drag contexts with `type` discrimination.

---

## Issue 4 — No Board Overview / Dashboard

**Severity:** High — users with multiple boards lose board-switching context.

**What happens:**  
[`web/app/(app)/boards/page.tsx:16-20`](web/app/(app)/boards/page.tsx) immediately redirects to the first board as soon as boards load:
```tsx
useEffect(() => {
  if (!loading && boards.length > 0) {
    router.replace(`/boards/${boards[0].id}/`);
  }
}, [boards, loading, router]);
```

There is **no board list / dashboard view**. The sidebar shows board names but provides no metadata (last updated, card count, collaborators). Users can't get an at-a-glance status of all their work.

**Fix recommendation:**  
- Remove the auto-redirect to first board.  
- Build a real `/boards` landing page showing a grid/list of board cards with: name, card count, collaborators, last-updated timestamp.
- This page is the natural home for the "+ New Board" CTA, not buried in the sidebar.

---

## Issue 5 — Cards Have No Labels, Priority, or Due Date

**Severity:** Medium — the data model is too thin for real task management.

**What's missing:**  
The `Card` type ([`web/lib/boards.ts:22-29`](web/lib/boards.ts)) only has `title`, `description`, `position`, and `attachments`. There are no:
- **Labels / tags** (e.g., "bug", "feature", "blocked")
- **Priority levels** (critical → low)
- **Due dates**
- **Assignee**

Without these, every card looks identical on the board — users cannot scan the board for status at a glance, which is the primary value proposition of a Kanban board.

**Fix recommendation:**  
- Add `labels: string[]`, `dueDate?: string`, and `priority?: "critical"|"high"|"medium"|"low"` to the `Card` type.  
- Extend the DB schema (a `JSONB`-style column or additional indexed columns), the API PATCH path, and render colored label chips and a priority badge on each card tile in the board.

---

## Issue 6 — Board Delete Uses `window.confirm()` — No Sharing UI

**Severity:** Medium (two sub-issues grouped by UX pattern quality).

**6a — `window.confirm()` for destructive actions**  
[`web/components/board/BoardView.tsx:247`](web/components/board/BoardView.tsx):
```tsx
if (!window.confirm(`Delete board "${board.name}"?`)) {
```
`window.confirm` is a browser-native modal with no styling, no branding, and different behavior across browsers. It blocks the JS thread and feels broken on mobile. This pattern should be replaced with a proper in-app confirmation modal (a small dialog with a red "Delete" button and a "Cancel" button).

**6b — No sharing / collaboration UI**  
The data model has `BoardPermission` (`"owner" | "write" | "read"`) and the sidebar already renders a "Shared with me" section — but there is **no UI to invite a collaborator** or manage permissions on a board. Part 6 of the plan specifies a "Sharing Dialog" but it is not yet built.

**Fix recommendation:**  
- Replace `window.confirm` with an in-app `<ConfirmModal>` component reusable across the app.  
- Add a **Share** button to the board header (next to Delete) that opens a dialog where the owner can enter a username/email and choose read/write access.

---

## Issue 7 — No Keyboard Navigation or Escape-to-Close in Card Modal

**Severity:** Medium — keyboard users and power users are blocked.

**What's missing:**  
- `CardModal` ([`web/components/board/CardModal.tsx`](web/components/board/CardModal.tsx)) has **no `onKeyDown` handler** — pressing `Escape` does not close it. This is a universal UX expectation (all major modals close on Escape). The board title input does have Escape (line 305–308) but the modal does not.  
- There are no keyboard shortcuts anywhere (no `N` to add card, no `?` help overlay, no arrow-key navigation between cards).  
- Tab order inside the modal is not managed — focus is not trapped inside the modal when it's open, so users can Tab behind it.

**Code gap:** `CardModal` sets `autoFocus` is not even present — focus is not moved into the modal at all on open.

**Fix recommendation:**  
- Add `useEffect` in `CardModal` that calls `document.addEventListener('keydown', ...)` and closes on `Escape` — same pattern used in `BoardView.tsx:82-89`.  
- Add `autoFocus` to the title input in the modal.  
- Add a focus trap (`Tab` cycles within the modal while it's open).  
- Document a shortcut map for: `N` = new card, `C` = new column, `Escape` = close modal / cancel edit.

---

## Summary Table

| # | Issue | Severity | Effort |
|---|-------|----------|--------|
| 1 | Drag-and-drop broken (React 19 / dnd incompatibility) | Critical | Medium — library swap |
| 2 | No card delete or move-to-column in modal | High | Low–Medium |
| 3 | Columns can't be renamed, deleted, or reordered | High | Medium |
| 4 | No board overview dashboard — auto-redirects away | High | Medium |
| 5 | Cards lack labels, priority, and due dates | Medium | High (schema + API + UI) |
| 6 | `window.confirm` for delete + no sharing UI | Medium | Low (confirm) / High (sharing) |
| 7 | No Escape-to-close modal, no keyboard shortcuts, no focus trap | Medium | Low |

---

## Quick-Win Recommendations (Highest ROI, lowest effort)

1. **Fix drag-and-drop** — unblock the core workflow. Pin React 18 or migrate to `dnd-kit`.  
2. **Add Escape-to-close in CardModal** — 5 lines of code, immediate quality-of-life improvement.  
3. **Add Delete card button in modal** — removes a dead-end user state.  
4. **Replace `window.confirm`** with a styled confirmation dialog — removes the jarring UX interruption.

