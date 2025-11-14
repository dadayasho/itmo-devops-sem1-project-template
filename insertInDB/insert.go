package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"

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
		_, err := pool.Exec(ctx, `
            INSERT INTO prices (id, name, category, price, create_date)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (id) DO UPDATE SET
                name = EXCLUDED.name,
                category = EXCLUDED.category,
                price = EXCLUDED.price,
                create_date = EXCLUDED.create_date
        `, rec[0], rec[1], rec[2], rec[3], rec[4])
		if err != nil {
			return fmt.Errorf("failed to insert record %v: %w", rec, err)
		}
	}
	return nil
}

func main() {
	db, error := database.СonnectDB()
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
