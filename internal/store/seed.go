package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/mrkiz-git/kanba-go/internal/domain"
)

func SeedAdmin(ctx context.Context, users *UserStore, email, passwordHash, name string) error {
	user, _, err := users.GetByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = users.Create(ctx, email, passwordHash, name, domain.RoleAdmin)
		return err
	}
	if err != nil {
		return err
	}
	return users.UpdateCredentials(ctx, user.ID, passwordHash, name)
}
