package entities

import "time"

type Product struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	SellerID    int       `json:"seller_id"`
	Images      []Image   `json:"images,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Image struct {
	ID        int    `json:"id"`
	ProductID int    `json:"product_id"`
	ImageURL  string `json:"image_url"`
}

type PinnedProduct struct {
	ID              int       `json:"id"`
	ProductID       int       `json:"product_id"`
	SellerID        int       `json:"seller_id"`
	SimilarityScore float64   `json:"similarity_score"`
	IsPinned        bool      `json:"is_pinned"`
	PinnedAt        time.Time `json:"pinned_at"`
	Product         *Product  `json:"product,omitempty"`
}