package database

import (
	"context"
	"live-shopping-ai/backend/internal/domain/entities"
	"live-shopping-ai/backend/internal/domain/repositories"

	"github.com/jackc/pgx/v5"
)

type postgresProductRepository struct {
	db *pgx.Conn
}

func NewPostgresProductRepository(db *pgx.Conn) repositories.ProductRepository {
	return &postgresProductRepository{db: db}
}

func (r *postgresProductRepository) FindAll(ctx context.Context) ([]entities.Product, error) {
	query := `
		SELECT p.id, p.name, p.description, p.price, p.seller_id, p.created_at, p.updated_at,
		       COALESCE(array_agg(i.image_url) FILTER (WHERE i.image_url IS NOT NULL), '{}') as image_urls
		FROM products p
		LEFT JOIN images i ON p.id = i.product_id
		GROUP BY p.id, p.name, p.description, p.price, p.seller_id, p.created_at, p.updated_at
		ORDER BY p.created_at DESC
	`
	
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []entities.Product
	for rows.Next() {
		var p entities.Product
		var imageURLs []string
		
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Price, &p.SellerID, &p.CreatedAt, &p.UpdatedAt, &imageURLs,
		)
		if err != nil {
			return nil, err
		}
		
		for _, url := range imageURLs {
			if url != "" {
				p.Images = append(p.Images, entities.Image{ImageURL: url})
			}
		}
		
		products = append(products, p)
	}

	return products, nil
}

func (r *postgresProductRepository) FindByID(ctx context.Context, id int) (*entities.Product, error) {
	query := `
		SELECT p.id, p.name, p.description, p.price, p.seller_id, p.created_at, p.updated_at
		FROM products p
		WHERE p.id = $1
	`
	
	var p entities.Product
	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.Price, &p.SellerID, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	
	imageQuery := `SELECT id, image_url FROM images WHERE product_id = $1`
	imageRows, err := r.db.Query(ctx, imageQuery, id)
	if err != nil {
		return nil, err
	}
	defer imageRows.Close()
	
	for imageRows.Next() {
		var img entities.Image
		imageRows.Scan(&img.ID, &img.ImageURL)
		img.ProductID = id
		p.Images = append(p.Images, img)
	}
	
	return &p, nil
}

func (r *postgresProductRepository) Create(ctx context.Context, product *entities.Product) error {
	query := `
		INSERT INTO products (name, description, price, seller_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	
	return r.db.QueryRow(
		ctx, query,
		product.Name, product.Description, product.Price, product.SellerID,
	).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)
}

func (r *postgresProductRepository) Update(ctx context.Context, product *entities.Product) error {
	query := `
		UPDATE products 
		SET name = $1, description = $2, price = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
		RETURNING updated_at
	`
	
	return r.db.QueryRow(
		ctx, query,
		product.Name, product.Description, product.Price, product.ID,
	).Scan(&product.UpdatedAt)
}

func (r *postgresProductRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM products WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *postgresProductRepository) FindBySellerID(ctx context.Context, sellerID int) ([]entities.Product, error) {
	query := `
		SELECT p.id, p.name, p.description, p.price, p.seller_id, p.created_at, p.updated_at,
		       COALESCE(array_agg(i.image_url) FILTER (WHERE i.image_url IS NOT NULL), '{}') as image_urls
		FROM products p
		LEFT JOIN images i ON p.id = i.product_id
		WHERE p.seller_id = $1
		GROUP BY p.id, p.name, p.description, p.price, p.seller_id, p.created_at, p.updated_at
		ORDER BY p.created_at DESC
	`
	
	rows, err := r.db.Query(ctx, query, sellerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []entities.Product
	for rows.Next() {
		var p entities.Product
		var imageURLs []string
		
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Price, &p.SellerID, &p.CreatedAt, &p.UpdatedAt, &imageURLs,
		)
		if err != nil {
			return nil, err
		}
		
		for _, url := range imageURLs {
			if url != "" {
				p.Images = append(p.Images, entities.Image{ImageURL: url})
			}
		}
		
		products = append(products, p)
	}

	return products, nil
}

func (r *postgresProductRepository) AddImages(ctx context.Context, productID int, imageURLs []string) error {
	for _, url := range imageURLs {
		query := `INSERT INTO images (product_id, image_url) VALUES ($1, $2)`
		_, err := r.db.Exec(ctx, query, productID, url)
		if err != nil {
			return err
		}
	}
	return nil
}