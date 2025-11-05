package database

import (
	"context"
	"live-shopping-ai/backend/internal/domain/entities"
	"live-shopping-ai/backend/internal/domain/repositories"

	"github.com/jackc/pgx/v5"
)

type postgresPinnedRepository struct {
	db *pgx.Conn
}

func NewPostgresPinnedRepository(db *pgx.Conn) repositories.PinnedProductRepository {
	return &postgresPinnedRepository{db: db}
}

func (r *postgresPinnedRepository) FindPinnedBySellerID(ctx context.Context, sellerID string) ([]entities.PinnedProduct, error) {
	query := `
		SELECT pp.id, pp.product_id, pp.seller_id, pp.similarity_score, pp.is_pinned, pp.pinned_at,
		       p.name, p.description, p.price
		FROM pinned_products pp
		JOIN products p ON pp.product_id = p.id
		WHERE pp.seller_id = $1 AND pp.is_pinned = true
		ORDER BY pp.pinned_at DESC
	`
	
	rows, err := r.db.Query(ctx, query, sellerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pinnedProducts []entities.PinnedProduct
	for rows.Next() {
		var pp entities.PinnedProduct
		var p entities.Product
		
		err := rows.Scan(
			&pp.ID, &pp.ProductID, &pp.SellerID, &pp.SimilarityScore, &pp.IsPinned, &pp.PinnedAt,
			&p.Name, &p.Description, &p.Price,
		)
		if err != nil {
			return nil, err
		}
		
		p.ID = pp.ProductID
		pp.Product = &p
		pinnedProducts = append(pinnedProducts, pp)
	}

	return pinnedProducts, nil
}

func (r *postgresPinnedRepository) PinProduct(ctx context.Context, pinData *entities.PinnedProduct) error {
	unpinQuery := `UPDATE pinned_products SET is_pinned = false WHERE seller_id = $1 AND is_pinned = true`
	_, err := r.db.Exec(ctx, unpinQuery, pinData.SellerID)
	if err != nil {
		return err
	}

	pinQuery := `
		INSERT INTO pinned_products (product_id, seller_id, similarity_score, is_pinned, pinned_at)
		VALUES ($1, $2, $3, true, CURRENT_TIMESTAMP)
		ON CONFLICT (product_id, seller_id) 
		DO UPDATE SET similarity_score = $3, is_pinned = true, pinned_at = CURRENT_TIMESTAMP
		RETURNING id
	`
	
	return r.db.QueryRow(ctx, pinQuery, pinData.ProductID, pinData.SellerID, pinData.SimilarityScore).Scan(&pinData.ID)
}

func (r *postgresPinnedRepository) UnpinProduct(ctx context.Context, productID int, sellerID string) error {
	query := `UPDATE pinned_products SET is_pinned = false WHERE product_id = $1 AND seller_id = $2`
	_, err := r.db.Exec(ctx, query, productID, sellerID)
	return err
}

func (r *postgresPinnedRepository) UnpinAllProducts(ctx context.Context, sellerID string) (int64, error) {
	query := `UPDATE pinned_products SET is_pinned = false WHERE seller_id = $1 AND is_pinned = true`
	result, err := r.db.Exec(ctx, query, sellerID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}