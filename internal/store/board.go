package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/google/uuid"
	"github.com/mrkiz-git/kanba-go/internal/domain"
)

var defaultColumnTitles = []string{"To Do", "In Progress", "Done"}

type BoardStore struct {
	db *sql.DB
}

func NewBoardStore(db *sql.DB) *BoardStore {
	return &BoardStore{db: db}
}

func (s *BoardStore) Create(ctx context.Context, name, ownerID string) (*domain.Board, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("board name required")
	}

	now := time.Now().UTC()
	boardID := uuid.New().String()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO boards (id, owner_id, name, version, created_at, updated_at)
		 VALUES (?, ?, ?, 1, ?, ?)`,
		boardID, ownerID, name, formatTime(now), formatTime(now),
	)
	if err != nil {
		if IsDuplicateBoardName(err) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("insert board: %w", err)
	}

	columns := make([]domain.Column, len(defaultColumnTitles))
	for i, title := range defaultColumnTitles {
		colID := uuid.New().String()
		_, err = tx.ExecContext(ctx,
			`INSERT INTO columns (id, board_id, title, position, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			colID, boardID, title, i, formatTime(now), formatTime(now),
		)
		if err != nil {
			return nil, fmt.Errorf("insert column: %w", err)
		}
		columns[i] = domain.Column{ID: colID, Title: title, Position: i, Cards: []domain.Card{}}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &domain.Board{
		ID:        boardID,
		Name:      name,
		Version:   1,
		Columns:   columns,
		UpdatedAt: now,
	}, nil
}

func (s *BoardStore) GetByID(ctx context.Context, id string) (*domain.Board, error) {
	return loadBoard(ctx, s.db, id)
}

func (s *BoardStore) ListForUser(ctx context.Context, userID string) ([]domain.BoardSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT b.id, b.name, b.version, b.updated_at, 'owner' AS permission
		FROM boards b
		WHERE b.owner_id = ?
		UNION ALL
		SELECT b.id, b.name, b.version, b.updated_at, bs.permission
		FROM board_shares bs
		JOIN boards b ON b.id = bs.board_id
		WHERE bs.user_id = ?
		ORDER BY updated_at DESC`, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []domain.BoardSummary
	for rows.Next() {
		var summary domain.BoardSummary
		var permission, updatedAt string
		if err := rows.Scan(&summary.ID, &summary.Name, &summary.Version, &updatedAt, &permission); err != nil {
			return nil, err
		}
		summary.Permission = domain.BoardPermission(permission)
		summary.UpdatedAt, err = parseTime(updatedAt)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}

func (s *BoardStore) Update(ctx context.Context, board *domain.Board) error {
	if board == nil {
		return fmt.Errorf("board required")
	}

	now := time.Now().UTC()
	expectedVersion := board.Version

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	newVersion, err := bumpBoardVersion(ctx, tx, board.ID, expectedVersion, board.Name, now)
	if err != nil {
		return err
	}

	if err := syncBoardContent(ctx, tx, board.ID, board.Columns, now); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	board.Version = newVersion
	board.UpdatedAt = now
	return nil
}

func (s *BoardStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM boards WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *BoardStore) ApplyPatch(ctx context.Context, boardID string, expectedVersion int, patch jsonpatch.Patch) (*domain.Board, error) {
	board, err := loadBoard(ctx, s.db, boardID)
	if err != nil {
		return nil, err
	}
	if board.Version != expectedVersion {
		return nil, ErrConflict
	}

	raw, err := json.Marshal(board)
	if err != nil {
		return nil, fmt.Errorf("marshal board: %w", err)
	}

	patched, err := patch.Apply(raw)
	if err != nil {
		return nil, fmt.Errorf("apply patch: %w", err)
	}

	var updated domain.Board
	if err := json.Unmarshal(patched, &updated); err != nil {
		return nil, fmt.Errorf("unmarshal patched board: %w", err)
	}
	updated.ID = boardID

	now := time.Now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	newVersion, err := bumpBoardVersion(ctx, tx, boardID, expectedVersion, updated.Name, now)
	if err != nil {
		return nil, err
	}

	if err := syncBoardContent(ctx, tx, boardID, updated.Columns, now); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	updated.Version = newVersion
	updated.UpdatedAt = now
	return &updated, nil
}

func (s *BoardStore) ResolvePermission(ctx context.Context, userID string, role domain.UserRole, boardID string) (domain.BoardPermission, error) {
	if role == domain.RoleAdmin {
		return domain.PermissionOwner, nil
	}

	var ownerID string
	err := s.db.QueryRowContext(ctx, `SELECT owner_id FROM boards WHERE id = ?`, boardID).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if ownerID == userID {
		return domain.PermissionOwner, nil
	}

	var permission string
	err = s.db.QueryRowContext(ctx,
		`SELECT permission FROM board_shares WHERE board_id = ? AND user_id = ?`,
		boardID, userID,
	).Scan(&permission)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrForbidden
	}
	if err != nil {
		return "", err
	}
	return domain.BoardPermission(permission), nil
}

func (s *BoardStore) Share(ctx context.Context, boardID, userID string, permission domain.SharePermission) error {
	var ownerID string
	err := s.db.QueryRowContext(ctx, `SELECT owner_id FROM boards WHERE id = ?`, boardID).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if ownerID == userID {
		return ErrForbidden
	}

	now := time.Now().UTC()
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO board_shares (id, board_id, user_id, permission, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		uuid.New().String(), boardID, userID, string(permission), formatTime(now),
	)
	if err != nil {
		if IsDuplicateShare(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (s *BoardStore) RevokeShare(ctx context.Context, boardID, userID string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM board_shares WHERE board_id = ? AND user_id = ?`,
		boardID, userID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *BoardStore) ListShares(ctx context.Context, boardID string) ([]domain.BoardShare, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT bs.id, bs.board_id, bs.user_id, u.email, bs.permission, bs.created_at
		FROM board_shares bs
		JOIN users u ON u.id = bs.user_id
		WHERE bs.board_id = ?
		ORDER BY bs.created_at ASC`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []domain.BoardShare
	for rows.Next() {
		var share domain.BoardShare
		var permission, createdAt string
		if err := rows.Scan(&share.ID, &share.BoardID, &share.UserID, &share.UserEmail, &permission, &createdAt); err != nil {
			return nil, err
		}
		share.Permission = domain.SharePermission(permission)
		share.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			return nil, err
		}
		shares = append(shares, share)
	}
	return shares, rows.Err()
}

func (s *BoardStore) GetBoardIDForCard(ctx context.Context, cardID string) (string, error) {
	var boardID string
	err := s.db.QueryRowContext(ctx, `
		SELECT b.id
		FROM cards c
		JOIN columns col ON col.id = c.column_id
		JOIN boards b ON b.id = col.board_id
		WHERE c.id = ?`, cardID).Scan(&boardID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return boardID, err
}

func (s *BoardStore) GetBoardIDForAttachment(ctx context.Context, attachmentID string) (string, error) {
	var boardID string
	err := s.db.QueryRowContext(ctx, `
		SELECT b.id
		FROM attachments a
		JOIN cards c ON c.id = a.card_id
		JOIN columns col ON col.id = c.column_id
		JOIN boards b ON b.id = col.board_id
		WHERE a.id = ?`, attachmentID).Scan(&boardID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return boardID, err
}

func IsDuplicateBoardName(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") && strings.Contains(msg, "boards")
}

func IsDuplicateShare(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") && strings.Contains(msg, "board_shares")
}

func loadBoard(ctx context.Context, db querier, id string) (*domain.Board, error) {
	var board domain.Board
	var updatedAt string
	err := db.QueryRowContext(ctx,
		`SELECT id, name, version, updated_at FROM boards WHERE id = ?`, id,
	).Scan(&board.ID, &board.Name, &board.Version, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	board.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return nil, err
	}

	colRows, err := db.QueryContext(ctx,
		`SELECT id, title, position FROM columns WHERE board_id = ? ORDER BY position ASC`, id)
	if err != nil {
		return nil, err
	}
	defer colRows.Close()

	columns := make([]domain.Column, 0)
	columnIDs := make([]string, 0)
	for colRows.Next() {
		var col domain.Column
		if err := colRows.Scan(&col.ID, &col.Title, &col.Position); err != nil {
			return nil, err
		}
		col.Cards = []domain.Card{}
		columns = append(columns, col)
		columnIDs = append(columnIDs, col.ID)
	}
	if err := colRows.Err(); err != nil {
		return nil, err
	}

	if len(columnIDs) > 0 {
		cardRows, err := db.QueryContext(ctx, `
			SELECT id, column_id, title, description, position, updated_at
			FROM cards
			WHERE column_id IN (`+placeholders(len(columnIDs))+`)
			ORDER BY column_id, position ASC`, argsFromStrings(columnIDs)...)
		if err != nil {
			return nil, err
		}
		defer cardRows.Close()

		cardsByColumn := make(map[string][]domain.Card)
		cardIDs := make([]string, 0)
		for cardRows.Next() {
			var card domain.Card
			var columnID, updatedAt string
			if err := cardRows.Scan(&card.ID, &columnID, &card.Title, &card.Description, &card.Position, &updatedAt); err != nil {
				return nil, err
			}
			card.UpdatedAt, err = parseTime(updatedAt)
			if err != nil {
				return nil, err
			}
			card.Attachments = []domain.Attachment{}
			cardsByColumn[columnID] = append(cardsByColumn[columnID], card)
			cardIDs = append(cardIDs, card.ID)
		}
		if err := cardRows.Err(); err != nil {
			return nil, err
		}

		attachmentsByCard, err := loadAttachments(ctx, db, cardIDs)
		if err != nil {
			return nil, err
		}

		for i := range columns {
			cards := cardsByColumn[columns[i].ID]
			if cards == nil {
				cards = []domain.Card{}
			}
			for j := range cards {
				attachments := attachmentsByCard[cards[j].ID]
				if attachments == nil {
					attachments = []domain.Attachment{}
				}
				cards[j].Attachments = attachments
			}
			columns[i].Cards = cards
		}
	}

	board.Columns = columns
	return &board, nil
}

func loadAttachments(ctx context.Context, db querier, cardIDs []string) (map[string][]domain.Attachment, error) {
	result := make(map[string][]domain.Attachment)
	if len(cardIDs) == 0 {
		return result, nil
	}

	rows, err := db.QueryContext(ctx, `
		SELECT id, card_id, filename, mime_type, size_bytes, created_at
		FROM attachments
		WHERE card_id IN (`+placeholders(len(cardIDs))+`)
		ORDER BY created_at ASC`, argsFromStrings(cardIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var att domain.Attachment
		var cardID, createdAt string
		if err := rows.Scan(&att.ID, &cardID, &att.Filename, &att.MimeType, &att.SizeBytes, &createdAt); err != nil {
			return nil, err
		}
		att.URL = "/api/attachments/" + att.ID
		att.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			return nil, err
		}
		result[cardID] = append(result[cardID], att)
	}
	return result, rows.Err()
}

func bumpBoardVersion(ctx context.Context, tx *sql.Tx, boardID string, expectedVersion int, name string, now time.Time) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("board name required")
	}

	res, err := tx.ExecContext(ctx,
		`UPDATE boards SET name = ?, version = version + 1, updated_at = ?
		 WHERE id = ? AND version = ?`,
		name, formatTime(now), boardID, expectedVersion,
	)
	if err != nil {
		if IsDuplicateBoardName(err) {
			return 0, ErrConflict
		}
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if n == 0 {
		var exists int
		err := tx.QueryRowContext(ctx, `SELECT 1 FROM boards WHERE id = ?`, boardID).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		if err != nil {
			return 0, err
		}
		return 0, ErrConflict
	}
	return expectedVersion + 1, nil
}

func syncBoardContent(ctx context.Context, tx *sql.Tx, boardID string, columns []domain.Column, now time.Time) error {
	existingCols, err := listColumnIDs(ctx, tx, boardID)
	if err != nil {
		return err
	}

	seenCols := make(map[string]struct{}, len(columns))
	for colIdx, col := range columns {
		colID := col.ID
		if colID == "" {
			colID = uuid.New().String()
		}
		seenCols[colID] = struct{}{}

		if containsString(existingCols, colID) {
			_, err = tx.ExecContext(ctx,
				`UPDATE columns SET title = ?, position = ?, updated_at = ? WHERE id = ? AND board_id = ?`,
				col.Title, colIdx, formatTime(now), colID, boardID,
			)
		} else {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO columns (id, board_id, title, position, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				colID, boardID, col.Title, colIdx, formatTime(now), formatTime(now),
			)
		}
		if err != nil {
			return err
		}

		if err := syncColumnCards(ctx, tx, colID, col.Cards, now); err != nil {
			return err
		}
	}

	for _, colID := range existingCols {
		if _, ok := seenCols[colID]; !ok {
			if _, err := tx.ExecContext(ctx, `DELETE FROM columns WHERE id = ? AND board_id = ?`, colID, boardID); err != nil {
				return err
			}
		}
	}
	return nil
}

func syncColumnCards(ctx context.Context, tx *sql.Tx, columnID string, cards []domain.Card, now time.Time) error {
	existingCards, err := listCardIDs(ctx, tx, columnID)
	if err != nil {
		return err
	}

	seenCards := make(map[string]struct{}, len(cards))
	for cardIdx, card := range cards {
		cardID := card.ID
		if cardID == "" {
			cardID = uuid.New().String()
		}
		seenCards[cardID] = struct{}{}

		if containsString(existingCards, cardID) {
			_, err = tx.ExecContext(ctx,
				`UPDATE cards SET column_id = ?, title = ?, description = ?, position = ?, updated_at = ?
				 WHERE id = ?`,
				columnID, card.Title, card.Description, cardIdx, formatTime(now), cardID,
			)
		} else {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO cards (id, column_id, title, description, position, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?)`,
				cardID, columnID, card.Title, card.Description, cardIdx, formatTime(now), formatTime(now),
			)
		}
		if err != nil {
			return err
		}
	}

	for _, cardID := range existingCards {
		if _, ok := seenCards[cardID]; !ok {
			if _, err := tx.ExecContext(ctx, `DELETE FROM cards WHERE id = ? AND column_id = ?`, cardID, columnID); err != nil {
				return err
			}
		}
	}
	return nil
}

func listColumnIDs(ctx context.Context, tx *sql.Tx, boardID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM columns WHERE board_id = ?`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func listCardIDs(ctx context.Context, tx *sql.Tx, columnID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM cards WHERE column_id = ?`, columnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ", ")
}

func argsFromStrings(values []string) []any {
	args := make([]any, len(values))
	for i, v := range values {
		args[i] = v
	}
	return args
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
