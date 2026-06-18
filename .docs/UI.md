# UI

Design specification for the Kanba frontend (Next.js static export). All visual decisions here are the single source of truth for Part 4 and beyond.

## Constraints

| Item | Decision |
|------|----------|
| Framework | Next.js (`output: 'export'`) served by Go at `/` |
| Styling | Tailwind CSS v4 utility classes; no CSS-in-JS |
| Dark mode | Light only (deferred) |
| Responsive target | Fully responsive — mobile, tablet, desktop |
| Drag and drop | `@hello-pangea/dnd` (maintained react-beautiful-dnd fork) |
| Markdown | `react-markdown` + `remark-gfm` for card description rendering |

---

## Design Tokens

### Typeface

System font stack — no web font download:

```css
font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
```

### Type Scale

| Token | Size | Weight | Usage |
|-------|------|--------|-------|
| `text-xs` | 0.75rem | 400 | Labels, attachment filenames, badges |
| `text-sm` | 0.875rem | 400 | Card body, sidebar items, table rows |
| `text-base` | 1rem | 400 | Default body |
| `text-lg` | 1.125rem | 600 | Column headers |
| `text-xl` | 1.25rem | 600 | Board title, page headings |
| `text-2xl` | 1.5rem | 700 | Auth page titles |

### Spacing

4px base unit. Use Tailwind's default scale: `1=4px, 2=8px, 3=12px, 4=16px, 6=24px, 8=32px, 12=48px, 16=64px`.

### Color Palette

| Token | Hex | Role |
|-------|-----|------|
| `slate-50` | `#f8fafc` | App background |
| `slate-100` | `#f1f5f9` | Sidebar background, input background, hover states |
| `slate-200` | `#e2e8f0` | Card borders, dividers, skeleton base |
| `slate-400` | `#94a3b8` | Placeholder text, secondary icons |
| `slate-600` | `#475569` | Secondary text, column card counts |
| `slate-900` | `#0f172a` | Primary text |
| `blue-50` | `#eff6ff` | Active sidebar item background |
| `blue-600` | `#2563eb` | Primary accent — buttons, links, focus rings |
| `blue-700` | `#1d4ed8` | Hover state on accent elements |
| `red-300` | `#fca5a5` | Error field border |
| `red-500` | `#ef4444` | Destructive actions, error toast background |
| `green-500` | `#22c55e` | Success toast, "active" status badge |
| `white` | `#ffffff` | Card surface, modal surface |

### Border Radius

| Token | Value | Usage |
|-------|-------|-------|
| `rounded` | 4px | Inputs, buttons, badges |
| `rounded-lg` | 8px | Cards, column headers |
| `rounded-xl` | 12px | Modals, dialogs |

### Shadows

| Context | Tailwind class |
|---------|---------------|
| Cards at rest | `shadow-sm` |
| Cards being dragged | `shadow-lg` |
| Modals | `shadow-2xl` + `backdrop-blur-sm` on overlay |
| Dropdown menus | `shadow-md` |

---

## Shell Layout

All authenticated routes render inside `AppShell`. Auth pages (`/login`, `/register`) render standalone.

```
AppShell
├── Sidebar (240px fixed, collapses to off-canvas drawer on mobile)
├── MainContent (flex-1, overflow-hidden, route outlet)
└── AiSidebar (320px right panel — placeholder slot, wired in Part 11)
```

### Sidebar

**Desktop (≥1024px):** Fixed 240px left column, `bg-slate-100`, full viewport height. Always visible.

**Mobile (<1024px):** Hidden off-canvas. Hamburger button (`≡`) in the topbar opens it as an overlay drawer.

Sidebar sections (top to bottom):

1. **Logo row** — app name "Kanba" in `text-xl font-bold text-slate-900`
2. **My Boards** — section label + list of owned `BoardListItem`s
3. **Shared with me** — section label + list of shared `BoardListItem`s (hidden if empty)
4. **+ New Board** — ghost button, opens inline name input
5. **Admin link** — visible only when `role === 'admin'`, links to `/admin`
6. **UserMenu** — pinned to bottom: avatar circle (initials), display name, "Sign out" on click

`BoardListItem` active state: `bg-blue-50 text-blue-700 font-medium`. Hover: `bg-slate-200`.

---

## Route → Component Map

| Route | Component | Shell |
|-------|-----------|-------|
| `/login` | `LoginPage` | None |
| `/register` | `RegisterPage` | None |
| `/boards` | `BoardListPage` (redirects to first board or shows empty state) | AppShell |
| `/boards/:id` | `BoardPage` | AppShell |
| `/admin` | Redirects to `/admin/users` | AppShell |
| `/admin/users` | `AdminUsersPage` | AppShell |
| `/admin/stats` | `AdminStatsPage` | AppShell |

Unauthenticated users hitting any protected route are redirected to `/login`.  
`/admin/*` routes additionally check `role === 'admin'` client-side; unauthorized users see a 403 message.

---

## Component Hierarchy

### BoardPage

```
BoardPage
├── BoardHeader
│   ├── BoardTitle (click-to-edit inline input)
│   ├── ShareButton (owner only) → SharingDialog
│   └── BoardMenu (⋯ dropdown: Rename always visible; Delete board visible to owner only)
├── ColumnScroller (horizontal scroll, flex row, gap-4, px-4 pb-4)
│   └── Column (repeating)
│       ├── ColumnHeader
│       │   ├── ColumnTitle (click-to-edit inline)
│       │   ├── CardCount badge (slate-600, text-xs)
│       │   └── ColumnMenu (⋯: Delete column)
│       ├── CardList (droppable zone)
│       │   └── CardItem (draggable)
│       │       ├── Title (text-sm font-medium)
│       │       ├── Description preview (1-2 lines truncated, Markdown stripped)
│       │       └── AttachmentCount badge (paperclip icon + count, hidden if 0)
│       └── AddCardButton ("+ Add a card", ghost style)
├── AddColumnButton ("+ Add column", shown at end of scroller)
└── CardModal (React portal, opens on CardItem click)
    ├── CardTitle (editable, text-xl)
    ├── CardDescription (toggle Edit / Preview; Preview renders Markdown)
    ├── AttachmentList
    │   └── AttachmentItem (filename, size, download icon, delete icon)
    └── AttachmentUpload (drag-and-drop zone + file picker)
```

### Admin Panel

```
AdminLayout (sub-layout inside AppShell)
├── AdminNav (horizontal tab bar: Users | Stats)
├── AdminUsersPage
│   ├── SearchInput (filter by email/name)
│   ├── UsersTable
│   │   └── UserRow: email, name, RoleBadge, StatusBadge, [Edit] button
│   ├── EditUserModal
│   │   ├── RoleSelect (admin / user)
│   │   └── StatusSelect (active / suspended)
│   └── DeleteUserConfirmDialog
└── AdminStatsPage
    ├── StatCard "Total users"
    └── StatCard "Total boards"
```

### Shared UI Primitives

| Component | Variants | Notes |
|-----------|----------|-------|
| `Button` | primary, secondary, destructive, ghost | Spinner replaces icon during in-flight requests; disabled during request |
| `Input` | default, error | Error state: `border-red-300`, red helper text below |
| `Textarea` | default, error | Same error pattern |
| `Select` | default | Native `<select>` wrapped for consistent styling |
| `Modal` | — | React portal; `shadow-2xl`; `rounded-xl`; backdrop `bg-black/40 backdrop-blur-sm` |
| `ConfirmDialog` | — | Wraps Modal; "Are you sure?" + Cancel + Confirm (destructive) |
| `Toast` | success, error, info | Top-right stack; auto-dismiss 4s; max 3 visible |
| `Badge` | role, status, permission | Color-coded: admin=`blue`, user=`slate`, active=`green`, suspended=`red` |
| `Avatar` | — | Initials fallback; colored by hash of name |
| `EmptyState` | — | Centered icon + message + optional CTA button |
| `Skeleton` | card, row | Animated pulse for loading states |

---

## Key Screen Wireframes

### Login / Register (standalone, no shell)

```
┌─────────────────────────────────────────────┐
│                                             │
│              Kanba                          │
│                                             │
│  ┌───────────────────────────────────────┐  │
│  │  Sign in to your account              │  │
│  │                                       │  │
│  │  Email ________________________________│  │
│  │  Password _____________________________│  │
│  │                                       │  │
│  │  [        Sign In         ]           │  │
│  │                                       │  │
│  │  Don't have an account? Register →    │  │
│  └───────────────────────────────────────┘  │
│                                             │
└─────────────────────────────────────────────┘
```

### Board View (desktop)

```
┌──────────┬──────────────────────────────────────────────────────────┐
│ Kanba    │  Sprint 12                          [Share] [⋯]          │
│──────────│──────────────────────────────────────────────────────────│
│ MY BOARDS│                                                          │
│ Sprint 12│  ┌───────────┐  ┌───────────┐  ┌───────────┐  ──►      │
│ Backlog  │  │ To Do  (3)│  │In Progress│  │ Done   (8)│            │
│──────────│  │───────────│  │───────────│  │───────────│            │
│ SHARED   │  │┌─────────┐│  │┌─────────┐│  │┌─────────┐│            │
│ Design Q │  ││ Card 1  ││  ││ Card A  ││  ││ Card X  ││            │
│──────────│  │└─────────┘│  │└─────────┘│  │└─────────┘│            │
│          │  │┌─────────┐│  │┌─────────┐│  │┌─────────┐│            │
│ + New    │  ││ Card 2  ││  ││ Card B  ││  ││ Card Y  ││            │
│   Board  │  │└─────────┘│  │└─────────┘│  │└─────────┘│            │
│──────────│  │           │  │           │  │           │            │
│          │  │+ Add card │  │+ Add card │  │+ Add card │            │
│ ⚙ Admin  │  └───────────┘  └───────────┘  └───────────┘            │
│ ○ Mark   │  + Add column                                            │
│   Logout │                                                          │
└──────────┴──────────────────────────────────────────────────────────┘
```

### Board View (mobile)

```
┌─────────────────────────────┐
│ ≡  Sprint 12      [Share]  │
│─────────────────────────────│
│  ┌──────────┐ ┌──────────┐ │◄── horizontal scroll
│  │ To Do (3)│ │In Progress│ │
│  │──────────│ │──────────│ │
│  │┌────────┐│ │┌────────┐│ │
│  ││ Card 1 ││ ││ Card A ││ │
│  │└────────┘│ │└────────┘│ │
│  │+ Add    │ │+ Add    │ │
│  └──────────┘ └──────────┘ │
└─────────────────────────────┘
```

### Card Modal

```
┌────────────────────────────────────────────────────┐
│  Login bug fix                              [✕]    │
│────────────────────────────────────────────────────│
│  Description                        [Edit][Preview]│
│  ┌──────────────────────────────────────────────┐  │
│  │  Markdown rendered here…                     │  │
│  └──────────────────────────────────────────────┘  │
│                                                    │
│  Attachments                                       │
│  ┌──────────────────────────────────────────────┐  │
│  │  screenshot.png   48 KB         [↓] [delete] │  │
│  └──────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────┐  │
│  │  Drop files here or click to upload          │  │
│  └──────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────┘
```

### Sharing Dialog (owner only)

```
┌───────────────────────────────────────────┐
│  Share "Sprint 12"                 [✕]   │
│───────────────────────────────────────────│
│  Invite by email                          │
│  ┌──────────────────────────┐ [Read  ▾]  │
│  │  collaborator@…          │  [Invite]  │
│  └──────────────────────────┘            │
│                                           │
│  People with access                       │
│  ┌───────────────────────────────────────┐│
│  │ ○ alice@co.com   Write    [Remove]    ││
│  │ ○ bob@co.com     Read     [Remove]    ││
│  └───────────────────────────────────────┘│
└───────────────────────────────────────────┘
```

### Admin — Users Table

```
┌──────────┬────────────────────────────────────────────────────────┐
│ Kanba    │  Admin Panel                                           │
│──────────│────────────────────────────────────────────────────────│
│ MY BOARDS│  [Users]  Stats                                        │
│ ...      │  ──────────────────────────────────────────────────    │
│──────────│  Search ___________________                            │
│ ⚙ Admin  │                                                        │
│          │  Email             Name    Role   Status    Actions    │
│          │  alice@co.com      Alice   admin  active    [Edit]     │
│          │  bob@co.com        Bob     user   active    [Edit]     │
│          │  carol@co.com      Carol   user   suspended [Edit]     │
└──────────┴────────────────────────────────────────────────────────┘
```

---

## Interaction Patterns

### Drag and Drop

- Cards draggable within and between columns via `@hello-pangea/dnd`.
- Columns reorderable by dragging their header drag handle.
- On drop: optimistic UI update fires immediately → `PATCH /api/boards/:id` with `If-Match: "<version>"`.
- On 409: silent re-fetch (`GET /api/boards/:id`) → recompute indices → retry patch. If retry also fails: revert UI + error toast.
- Cards hidden in `read` permission mode drag handles.

### Inline Editing

Board title and column titles are click-to-edit:
1. Click → input appears pre-filled with current value
2. Enter or blur → commit (PATCH)
3. Escape → cancel, restore original value
4. Empty value → rejected (validation inline, no request sent)

### Real-Time Sync (WebSocket)

- Subscribe to `GET /api/boards/:id/ws` on board mount; unsubscribe on unmount.
- Events from the current user's own actions are de-duped (already applied optimistically).
- Events from other users: apply to board state in React store + show "Board updated by another user" info toast.

### Loading States

| Context | Treatment |
|---------|-----------|
| Initial board load | Skeleton cards (animated pulse) in each column |
| Button action in-flight | Spinner replaces icon; button `disabled` |
| Attachment upload | Progress bar inside upload zone |
| Admin table load | Skeleton rows |

### Error Handling

| Scenario | Treatment |
|----------|-----------|
| Form validation failure | Red helper text below field; `border-red-300` on input |
| API error (4xx/5xx) | Error toast (top-right, 4s) with `error.message` from `APIError` envelope |
| 401 on any request | Redirect to `/login` |
| 409 on PATCH | Silent re-fetch + retry; error toast only if retry also fails |
| WS disconnect | Reconnect with exponential backoff (1s, 2s, 4s, max 30s); "Reconnecting…" banner |

### Permission-Aware UI

| Permission | Drag handles | Add card | Add column | Share button | Board menu |
|------------|-------------|----------|------------|--------------|------------|
| `read` | Hidden | Hidden | Hidden | Hidden | Hidden |
| `write` | Visible | Visible | Visible | Hidden | Rename only |
| `owner` | Visible | Visible | Visible | Visible | Full |
| `admin` (system) | Visible | Visible | Visible | Visible | Full |

---

## Responsive Breakpoints

| Breakpoint | Sidebar | Board columns |
|------------|---------|---------------|
| `<640px` (sm) | Off-canvas drawer | Horizontal scroll, ~280px col width |
| `640–1023px` (sm–lg) | Off-canvas drawer | Horizontal scroll, ~300px col width |
| `≥1024px` (lg) | Fixed 240px | Horizontal scroll, ~300px col width |

---

## Cross-References

- API endpoints: `API.md`
- Domain types: `BOARD_SCHEMA.md`
- Auth & permissions: `AUTH.md`
- Database: `DATABASE.md`
- MCP tools: `MCP.md`
