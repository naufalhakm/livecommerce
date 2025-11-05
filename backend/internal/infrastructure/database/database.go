package database

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

var DB *pgx.Conn

func InitDatabase() *pgx.Conn {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "host=localhost port=5432 user=postgres password=postgres dbname=livecommerce sslmode=disable"
	}
	
	
	config, err := pgx.ParseConfig(connStr)
	if err != nil {
		log.Fatal("Config parse error:", err)
	}
	
	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	
	DB, err = pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		log.Fatal("Database connection error:", err)
	}

	var version string
	if err := DB.QueryRow(context.Background(), "SELECT version()").Scan(&version); err != nil {
		log.Fatal("Database query error:", err)
	}

	createTables()
	
	return DB
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			price DECIMAL(10,2),
			seller_id INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS images (
			id SERIAL PRIMARY KEY,
			product_id INTEGER REFERENCES products(id) ON DELETE CASCADE,
			image_url TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS pinned_products (
			id SERIAL PRIMARY KEY,
			product_id INTEGER REFERENCES products(id) ON DELETE CASCADE,
			seller_id INTEGER NOT NULL,
			similarity_score DECIMAL(3,2),
			is_pinned BOOLEAN DEFAULT false,
			pinned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(product_id, seller_id)
		)`,
		`CREATE TABLE IF NOT EXISTS livestreams (
			id SERIAL PRIMARY KEY,
			seller_id VARCHAR(255) NOT NULL,
			seller_name VARCHAR(255) NOT NULL,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			is_live BOOLEAN DEFAULT true,
			viewer_count INTEGER DEFAULT 0,
			started_at TIMESTAMP NOT NULL,
			ended_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_livestreams_seller_id ON livestreams(seller_id)`,
		`CREATE INDEX IF NOT EXISTS idx_livestreams_is_live ON livestreams(is_live)`,
	}

	for _, query := range queries {
		if _, err := DB.Exec(context.Background(), query); err != nil {
		}
	}

}