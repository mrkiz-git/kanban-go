package handler

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/go-chi/chi/v5"
	"github.com/mrkiz-git/kanba-go/internal/auth"
	"github.com/mrkiz-git/kanba-go/internal/domain"
	"github.com/mrkiz-git/kanba-go/internal/store"
)

type BoardHandler struct {
	boards *store.BoardStore
	users  *store.UserStore
}

func NewBoardHandler(boards *store.BoardStore, users *store.UserStore) *BoardHandler {
	return &BoardHandler{boards: boards, users: users}
}

type createBoardRequest struct {
	Name string `json:"name"`
}

type shareRequest struct {
	Email      string                 `json:"email"`
	Permission domain.SharePermission `json:"permission"`
}

func (h *BoardHandler) List(w http.ResponseWriter, r *http.Request) {
	authUser, ok := auth.UserFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w)
		return
	}

	boards, err := h.boards.ListForUser(r.Context(), authUser.ID)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "internal_error", "internal error", nil)
		return
	}
	if boards == nil {
		boards = []domain.BoardSummary{}
	}
	WriteJSON(w, http.StatusOK, map[string]any{"boards": boards})
}

func (h *BoardHandler) Create(w http.ResponseWriter, r *http.Request) {
	authUser, ok := auth.UserFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w)
		return
	}

	var req createBoardRequest
	if err := jsonDecode(r, &req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "bad_request", "Malformed JSON", nil)
		return
	}

	name := strings.TrimSpace(req.Name)
	if err := validateBoardName(name); err != nil {
		WriteAPIError(w, http.StatusUnprocessableEntity, "validation_error", err.Error(), nil)
		return
	}

	board, err := h.boards.Create(r.Context(), name, authUser.ID)
	if err != nil {
		writeBoardStoreError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, board)
}

func (h *BoardHandler) Get(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "id")
	board, err := h.boards.GetByID(r.Context(), boardID)
	if err != nil {
		writeBoardStoreError(w, err)
		return
	}
	setBoardETag(w, board.Version)
	WriteJSON(w, http.StatusOK, board)
}

func (h *BoardHandler) Update(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "id")
	version, ok := parseIfMatch(r)
	if !ok {
		WriteAPIError(w, http.StatusBadRequest, "bad_request", "If-Match header required", nil)
		return
	}

	var board domain.Board
	if err := jsonDecode(r, &board); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "bad_request", "Malformed JSON", nil)
		return
	}

	name := strings.TrimSpace(board.Name)
	if err := validateBoardName(name); err != nil {
		WriteAPIError(w, http.StatusUnprocessableEntity, "validation_error", err.Error(), nil)
		return
	}
	if err := validateBoardColumns(board.Columns); err != nil {
		WriteAPIError(w, http.StatusUnprocessableEntity, "validation_error", err.Error(), nil)
		return
	}

	board.ID = boardID
	board.Name = name
	board.Version = version

	if err := h.boards.Update(r.Context(), &board); err != nil {
		writeBoardStoreError(w, err)
		return
	}

	updated, err := h.boards.GetByID(r.Context(), boardID)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "internal_error", "internal error", nil)
		return
	}
	setBoardETag(w, updated.Version)
	WriteJSON(w, http.StatusOK, updated)
}

func (h *BoardHandler) Patch(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "id")
	version, ok := parseIfMatch(r)
	if !ok {
		WriteAPIError(w, http.StatusBadRequest, "bad_request", "If-Match header required", nil)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "bad_request", "Malformed JSON", nil)
		return
	}

	patch, err := jsonpatch.DecodePatch(body)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "bad_request", "Invalid JSON Patch", nil)
		return
	}

	updated, err := h.boards.ApplyPatch(r.Context(), boardID, version, patch)
	if err != nil {
		writeBoardStoreError(w, err)
		return
	}

	setBoardETag(w, updated.Version)
	WriteJSON(w, http.StatusOK, updated)
}

func (h *BoardHandler) Delete(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "id")
	if err := h.boards.Delete(r.Context(), boardID); err != nil {
		writeBoardStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *BoardHandler) ListShares(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "id")
	shares, err := h.boards.ListShares(r.Context(), boardID)
	if err != nil {
		writeBoardStoreError(w, err)
		return
	}
	if shares == nil {
		shares = []domain.BoardShare{}
	}
	WriteJSON(w, http.StatusOK, map[string]any{"shares": shares})
}

func (h *BoardHandler) CreateShare(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "id")

	var req shareRequest
	if err := jsonDecode(r, &req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "bad_request", "Malformed JSON", nil)
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		WriteAPIError(w, http.StatusUnprocessableEntity, "validation_error", "Email is required", nil)
		return
	}
	if req.Permission != domain.SharePermissionRead && req.Permission != domain.SharePermissionWrite {
		WriteAPIError(w, http.StatusUnprocessableEntity, "validation_error", "Permission must be read or write", nil)
		return
	}

	user, _, err := h.users.GetByEmail(r.Context(), email)
	if errors.Is(err, sql.ErrNoRows) {
		WriteAPIError(w, http.StatusNotFound, "not_found", "User not found", nil)
		return
	}
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "internal_error", "internal error", nil)
		return
	}

	if err := h.boards.Share(r.Context(), boardID, user.ID, req.Permission); err != nil {
		writeBoardStoreError(w, err)
		return
	}

	shares, err := h.boards.ListShares(r.Context(), boardID)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "internal_error", "internal error", nil)
		return
	}

	for _, share := range shares {
		if share.UserID == user.ID {
			WriteJSON(w, http.StatusCreated, share)
			return
		}
	}
	WriteAPIError(w, http.StatusInternalServerError, "internal_error", "internal error", nil)
}

func (h *BoardHandler) RevokeShare(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		WriteAPIError(w, http.StatusBadRequest, "bad_request", "User ID required", nil)
		return
	}

	if err := h.boards.RevokeShare(r.Context(), boardID, userID); err != nil {
		writeBoardStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeBoardStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		WriteAPIError(w, http.StatusNotFound, "not_found", "Board not found", nil)
	case errors.Is(err, store.ErrConflict):
		WriteAPIError(w, http.StatusConflict, "conflict", "Conflict", nil)
	case errors.Is(err, store.ErrForbidden):
		WriteForbidden(w)
	default:
		WriteAPIError(w, http.StatusInternalServerError, "internal_error", "internal error", nil)
	}
}

func parseIfMatch(r *http.Request) (int, bool) {
	raw := strings.TrimSpace(r.Header.Get("If-Match"))
	raw = strings.Trim(raw, `"`)
	if raw == "" {
		return 0, false
	}
	version, err := strconv.Atoi(raw)
	if err != nil || version < 1 {
		return 0, false
	}
	return version, true
}

func setBoardETag(w http.ResponseWriter, version int) {
	w.Header().Set("ETag", strconv.Quote(strconv.Itoa(version)))
}

func validateBoardName(name string) error {
	if len(name) < 1 || len(name) > 100 {
		return errors.New("Board name must be 1–100 characters")
	}
	return nil
}

func validateBoardColumns(columns []domain.Column) error {
	if len(columns) < 1 || len(columns) > 20 {
		return errors.New("Board must have 1–20 columns")
	}
	for _, col := range columns {
		title := strings.TrimSpace(col.Title)
		if len(title) < 1 || len(title) > 100 {
			return errors.New("Column title must be 1–100 characters")
		}
		if len(col.Cards) > 500 {
			return errors.New("Column cannot have more than 500 cards")
		}
		for _, card := range col.Cards {
			cardTitle := strings.TrimSpace(card.Title)
			if len(cardTitle) < 1 || len(cardTitle) > 200 {
				return errors.New("Card title must be 1–200 characters")
			}
			if len(card.Description) > 10000 {
				return errors.New("Card description must be at most 10,000 characters")
			}
		}
	}
	return nil
}
