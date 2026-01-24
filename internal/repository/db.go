package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(db_url string, ctx context.Context) (*PostgresStore, error) {

	pool, err := pgxpool.New(ctx, db_url)
	if err != nil {
		return nil, err
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, err
	}
	ps := new(PostgresStore)
	ps.pool = pool
	return ps, err
}

func (ps *PostgresStore) InitPostgresStore(ctx context.Context) error {
	query := `create table if not exists urls (
        original_url varchar(255) primary key,
		shortened_url varchar(8),
        created_at timestamp default current_timestamp
    )`
	_, err := ps.pool.Exec(ctx, query)
	return err
}

func (ps *PostgresStore) Close() {
	ps.pool.Close()
}

func (ps *PostgresStore) StoreURL(shortenedURL string, originalURL string, ctx context.Context) error {
	var err error

	query := `insert into urls (original_url, shortened_url)
				  VALUES ($1, $2)`
	_, err = ps.pool.Exec(ctx, query, originalURL, shortenedURL)

	return err
}

func (ps *PostgresStore) GetURL(shortenedURL string, ctx context.Context) (string, error) {
	var originalURL string
	query := `select original_url from urls where shortened_url = $1`
	err := ps.pool.QueryRow(ctx, query, shortenedURL).Scan(&originalURL)
	return originalURL, err
}
