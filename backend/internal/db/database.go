package db

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

var DB *pgx.Conn

func InitDatabase() {
	// Get connection string from environment
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Printf("DATABASE_URL not set, using local fallback")
		connStr = "host=localhost port=5432 user=postgres password=postgres dbname=livecommerce sslmode=disable"
	}
	
	log.Printf("Connecting to database...")
	
	// Parse config and disable prepared statements for pooler
	config, err := pgx.ParseConfig(connStr)
	if err != nil {
		log.Fatal("Config parse error:", err)
	}
	
	// Disable prepared statements to avoid cache issues with pooler
	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	
	DB, err = pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		log.Fatal("Database connection error:", err)
	}

	// Test connection
	var version string
	if err := DB.QueryRow(context.Background(), "SELECT version()").Scan(&version); err != nil {
		log.Fatal("Database query error:", err)
	}

	log.Printf("✅ Connected to database: PostgreSQL")
	createTables()
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
	}

	for _, query := range queries {
		if _, err := DB.Exec(context.Background(), query); err != nil {
			log.Printf("Error creating table: %v", err)
		}
	}

	log.Println("Database tables created successfully")
}