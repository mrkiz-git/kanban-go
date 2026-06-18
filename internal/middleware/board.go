package middleware

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mrkiz-git/kanba-go/internal/auth"
	"github.com/mrkiz-git/kanba-go/internal/domain"
	"github.com/mrkiz-git/kanba-go/internal/handler"
	"github.com/mrkiz-git/kanba-go/internal/store"
)

type BoardPermConfig struct {
	Boards *store.BoardStore
	Min    domain.BoardPermission
}

func RequireBoardPerm(cfg BoardPermConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authUser, ok := auth.UserFromContext(r.Context())
			if !ok {
				handler.WriteUnauthorized(w)
				return
			}

			boardID := chi.URLParam(r, "id")
			if boardID == "" {
				handler.WriteAPIError(w, http.StatusBadRequest, "bad_request", "Board ID required", nil)
				return
			}

			perm, err := cfg.Boards.ResolvePermission(r.Context(), authUser.ID, authUser.Role, boardID)
			if errors.Is(err, store.ErrNotFound) {
				handler.WriteAPIError(w, http.StatusNotFound, "not_found", "Board not found", nil)
				return
			}
			if errors.Is(err, store.ErrForbidden) {
				handler.WriteForbidden(w)
				return
			}
			if err != nil {
				handler.WriteAPIError(w, http.StatusInternalServerError, "internal_error", "internal error", nil)
				return
			}
			if !permAtLeast(perm, cfg.Min) {
				handler.WriteForbidden(w)
				return
			}

			ctx := auth.WithBoardAccess(r.Context(), auth.BoardAccess{
				BoardID:    boardID,
				Permission: perm,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func permAtLeast(actual, required domain.BoardPermission) bool {
	rank := map[domain.BoardPermission]int{
		domain.PermissionRead:  1,
		domain.PermissionWrite: 2,
		domain.PermissionOwner: 3,
	}
	return rank[actual] >= rank[required]
}
