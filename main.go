package main

import (
	//"fmt"
	"log/slog"
	"os"

	//"math/rand"
	//"strconv"
	//"encoding/json"
	//"github.com/gorilla/mux"
	//"bytes"
	//"io"

	"itmo-devops-sem1-project-template/internal/config"
	database "itmo-devops-sem1-project-template/internal/db"
	//"itmo-devops-sem1-project-template/internal/tools"
)

const (
	envLocal = "local"
)

type InsertResponse struct {
	total_count      int `yaml: TotalCount`
	duplicates_count int `yaml: DuplicatesCount`
	total_items      int `yaml: DuplicatesCount`
	total_categories int `yaml: TotalCategories`
	total_price      int `yaml: TotalPrice`
}

func main() {
	cfg := config.MustLoad()
	//fmt.Println(cfg)
	log := setupLogger(cfg.Env)
	log.Info("starting server", slog.String("env", cfg.Env))

	db, error := database.СonnectDB()
	if error != nil {
		log.Error("DB connection failed",
			slog.String("env", cfg.Env),
			slog.String("error", error.Error()))

	} else {
		log.Info("Connetion to db is succes", slog.String("env", cfg.Env))
	}
	defer db.Close()

}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
	return log
}
