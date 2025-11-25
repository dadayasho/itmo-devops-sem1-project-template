package main

import (
	"context"
	"encoding/csv"
	"log"
	"os"
	"strconv"
	"time"

	database "itmo-devops-sem1-project-template/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

func importCSVWithUpsert(pool *pgxpool.Pool, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return err
	}

	ctx := context.Background()

	for _, rec := range records[1:] {
		price, err := strconv.ParseFloat(rec[3], 64)
		if err != nil {
			continue
		}
		date, err := time.Parse("2006-01-02", rec[4])
		if err != nil {
			continue
		}

		_, err = pool.Exec(ctx, `
        INSERT INTO prices (name, category, price, create_date)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (name, category, price, create_date) DO NOTHING;
    `, rec[1], rec[2], price, date)
		if err != nil {
			// обработка ошибки
		}
	}

	return nil
}

func main() {
	db, error := database.ConnectDB()
	if error != nil {
		log.Fatalf("DB connection lost: %v", error)
	}
	error = importCSVWithUpsert(db, "sample_data/data.csv")
	if error != nil {
		log.Fatalf("Failed to import CSV: %v", error)
	}
	log.Println("CSV import finished successfully")
	db.Close()
}
