package repositories

import (
	"context"
	"live-shopping-ai/backend/internal/domain/entities"
)

type ProductRepository interface {
	FindAll(ctx context.Context) ([]entities.Product, error)
	FindByID(ctx context.Context, id int) (*entities.Product, error)
	FindBySellerID(ctx context.Context, sellerID int) ([]entities.Product, error)
	Create(ctx context.Context, product *entities.Product) error
	Update(ctx context.Context, product *entities.Product) error
	Delete(ctx context.Context, id int) error
	AddImages(ctx context.Context, productID int, imageURLs []string) error
}

type PinnedProductRepository interface {
	FindPinnedBySellerID(ctx context.Context, sellerID string) ([]entities.PinnedProduct, error)
	PinProduct(ctx context.Context, pinData *entities.PinnedProduct) error
	UnpinProduct(ctx context.Context, productID int, sellerID string) error
	UnpinAllProducts(ctx context.Context, sellerID string) (int64, error)
}