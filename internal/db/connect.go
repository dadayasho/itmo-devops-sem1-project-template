package db

import (
	//"fmt"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var db *pgxpool.Pool

func connectDB() *pgxpool.Pool {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	user := os.Getenv("DB_USER_NAME")
	password := os.Getenv("DB_USER_PASSWORD")
	ip := os.Getenv("DB_IP")
	db_name := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")
	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, ip, db_name, port)
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		log.Fatalf("Unable to parse DB config: %v\n", err)
	}

	dbpool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	return dbpool
}
