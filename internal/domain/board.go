package domain

import "time"

type SharePermission string

const (
	SharePermissionWrite SharePermission = "write"
	SharePermissionRead  SharePermission = "read"
)

type BoardShare struct {
	ID         string          `json:"id"`
	BoardID    string          `json:"boardId"`
	UserID     string          `json:"userId"`
	UserEmail  string          `json:"userEmail"`
	Permission SharePermission `json:"permission"`
	CreatedAt  time.Time       `json:"createdAt"`
}

type BoardPermission string

const (
	PermissionOwner BoardPermission = "owner"
	PermissionWrite BoardPermission = "write"
	PermissionRead  BoardPermission = "read"
)

type BoardSummary struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Permission BoardPermission `json:"permission"`
	Version    int             `json:"version"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

type Board struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Version   int       `json:"version"`
	Columns   []Column  `json:"columns"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Column struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Position int    `json:"position"`
	Cards    []Card `json:"cards"`
}

type Card struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Position    int          `json:"position"`
	Attachments []Attachment `json:"attachments,omitempty"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

type Attachment struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	MimeType  string    `json:"mimeType"`
	SizeBytes int64     `json:"sizeBytes"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
}
