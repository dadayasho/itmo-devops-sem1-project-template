package database

import (
	//"fmt"
	"context"
	"fmt"

	//"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	//"github.com/joho/godotenv"
)

func СonnectDB() (*pgxpool.Pool, error) {
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	ip := os.Getenv("POSTGRES_HOST")
	db_name := os.Getenv("POSTGRES_DB")
	port := os.Getenv("POSTGRES_PORT")
	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, ip, port, db_name)
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("error parsing DB config: %w", err)
	}

	dbpool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("error connecting to DB: %w", err)
	}

	return dbpool, nil
}
