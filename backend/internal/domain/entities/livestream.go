package entities

import "time"

type LiveStream struct {
	ID          int       `json:"id" db:"id"`
	SellerID    string    `json:"seller_id" db:"seller_id"`
	SellerName  string    `json:"seller_name" db:"seller_name"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	IsLive      bool      `json:"is_live" db:"is_live"`
	ViewerCount int       `json:"viewer_count" db:"viewer_count"`
	StartedAt   time.Time `json:"started_at" db:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty" db:"ended_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type LiveStreamRequest struct {
	SellerID    string `json:"seller_id" binding:"required"`
	SellerName  string `json:"seller_name" binding:"required"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

type LiveStreamResponse struct {
	Success bool        `json:"success"`
	Data    *LiveStream `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

type LiveStreamListResponse struct {
	Success bool         `json:"success"`
	Data    []LiveStream `json:"data"`
	Message string       `json:"message,omitempty"`
}