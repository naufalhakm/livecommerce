package database

import (
	"context"
	"live-shopping-ai/backend/internal/domain/entities"
	"time"

	"github.com/jackc/pgx/v5"
)

type PostgresLiveStreamRepository struct{
		db *pgx.Conn
}

func NewPostgresLiveStreamRepository(db *pgx.Conn) *PostgresLiveStreamRepository {
	return &PostgresLiveStreamRepository{
		db: db,
	}
}

func (r *PostgresLiveStreamRepository) CreateLiveStream(stream *entities.LiveStream) error {
	query := `
		INSERT INTO livestreams (seller_id, seller_name, title, description, is_live, viewer_count, started_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`
	
	now := time.Now()
	return r.db.QueryRow(context.Background(), query,
		stream.SellerID,
		stream.SellerName,
		stream.Title,
		stream.Description,
		stream.IsLive,
		stream.ViewerCount,
		stream.StartedAt,
		now,
		now,
	).Scan(&stream.ID)
}

func (r *PostgresLiveStreamRepository) GetLiveStreamBySellerID(sellerID string) (*entities.LiveStream, error) {
	query := `
		SELECT id, seller_id, seller_name, title, description, is_live, viewer_count, started_at, ended_at, created_at, updated_at
		FROM livestreams 
		WHERE seller_id = $1 AND is_live = true
		ORDER BY started_at DESC 
		LIMIT 1`
	
	stream := &entities.LiveStream{}
	err := r.db.QueryRow(context.Background(), query, sellerID).Scan(
		&stream.ID,
		&stream.SellerID,
		&stream.SellerName,
		&stream.Title,
		&stream.Description,
		&stream.IsLive,
		&stream.ViewerCount,
		&stream.StartedAt,
		&stream.EndedAt,
		&stream.CreatedAt,
		&stream.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return stream, nil
}

func (r *PostgresLiveStreamRepository) GetActiveLiveStreams() ([]entities.LiveStream, error) {
	query := `
		SELECT id, seller_id, seller_name, title, description, is_live, viewer_count, started_at, ended_at, created_at, updated_at
		FROM livestreams 
		WHERE is_live = true
		ORDER BY started_at DESC`
	
	rows, err := r.db.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var streams []entities.LiveStream
	for rows.Next() {
		var stream entities.LiveStream
		err := rows.Scan(
			&stream.ID,
			&stream.SellerID,
			&stream.SellerName,
			&stream.Title,
			&stream.Description,
			&stream.IsLive,
			&stream.ViewerCount,
			&stream.StartedAt,
			&stream.EndedAt,
			&stream.CreatedAt,
			&stream.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		streams = append(streams, stream)
	}
	
	return streams, nil
}

func (r *PostgresLiveStreamRepository) UpdateLiveStreamStatus(sellerID string, isLive bool) error {
	query := `UPDATE livestreams SET is_live = $1, updated_at = $2 WHERE seller_id = $3 AND is_live = true`
	_, err := r.db.Exec(context.Background(), query, isLive, time.Now(), sellerID)
	return err
}

func (r *PostgresLiveStreamRepository) UpdateViewerCount(sellerID string, count int) error {
	query := `UPDATE livestreams SET viewer_count = $1, updated_at = $2 WHERE seller_id = $3 AND is_live = true`
	_, err := r.db.Exec(context.Background(), query, count, time.Now(), sellerID)
	return err
}

func (r *PostgresLiveStreamRepository) EndLiveStream(sellerID string) error {
	query := `UPDATE livestreams SET is_live = false, ended_at = $1, updated_at = $2 WHERE seller_id = $3 AND is_live = true`
	_, err := r.db.Exec(context.Background(), query, time.Now(), time.Now(), sellerID)
	return err
}