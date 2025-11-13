package database

import (
	//"fmt"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func СonnectDB(path string) (*pgxpool.Pool, error) {
	err := godotenv.Load(path)
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	user := os.Getenv("DB_USER_NAME")
	password := os.Getenv("DB_USER_PASSWORD")
	ip := os.Getenv("DB_IP")
	db_name := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")
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
