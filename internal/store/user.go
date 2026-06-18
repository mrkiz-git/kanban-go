package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mrkiz-git/kanba-go/internal/domain"
)

type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) Create(ctx context.Context, email, passwordHash, name string, role domain.UserRole) (*domain.User, error) {
	now := time.Now().UTC()
	id := uuid.New().String()

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, name, role, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 'active', ?, ?)`,
		id, email, passwordHash, name, string(role), formatTime(now), formatTime(now),
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return &domain.User{
		ID:        id,
		Email:     email,
		Name:      name,
		Role:      role,
		Status:    domain.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*domain.User, string, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, name, role, status, created_at, updated_at
		 FROM users WHERE email = ?`, email,
	)
	return scanUserWithHash(row)
}

func (s *UserStore) GetByID(ctx context.Context, id string) (*domain.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, name, role, status, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	)
	user, _, err := scanUserWithHash(row)
	return user, err
}

func (s *UserStore) UpdateCredentials(ctx context.Context, id, passwordHash, name string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ?, name = ?, updated_at = ? WHERE id = ?`,
		passwordHash, name, formatTime(time.Now().UTC()), id,
	)
	return err
}

func (s *UserStore) SetStatus(ctx context.Context, id string, status domain.UserStatus) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), formatTime(time.Now().UTC()), id)
	return err
}

func IsDuplicateEmail(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") && strings.Contains(msg, "email")
}

func scanUserWithHash(row *sql.Row) (*domain.User, string, error) {
	var u domain.User
	var role, status, passwordHash, createdAt, updatedAt string

	err := row.Scan(&u.ID, &u.Email, &passwordHash, &u.Name, &role, &status, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", sql.ErrNoRows
	}
	if err != nil {
		return nil, "", err
	}

	u.Role = domain.UserRole(role)
	u.Status = domain.UserStatus(status)
	u.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return nil, "", err
	}
	u.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return nil, "", err
	}

	return &u, passwordHash, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}
