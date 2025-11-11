package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	//"math/rand"
	//"strconv"
	//"encoding/json"

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
	//подгрузка конфига
	cfg := config.MustLoad()
	log := setupLogger(cfg.Env)
	log.Info("got config", slog.String("env", cfg.Env))

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, Go!")
	})

	// получение данных из конфига
	srv := &http.Server{
		Addr:        cfg.Address,
		ReadTimeout: cfg.Timeout,
		IdleTimeout: cfg.IdleTimeout,
		Handler:     mux,
	}

	//открытие содеинения с бд
	db, error := database.СonnectDB()
	if error != nil {
		log.Error("DB connection failed",
			slog.String("env", cfg.Env),
			slog.String("error", error.Error()))

	} else {
		log.Info("Connetion to db is success", slog.String("env", cfg.Env))
	}
	defer db.Close()

	//это запуск сервера
	log.Info("server listening", slog.String("addr", cfg.Address))
	if err := srv.ListenAndServe(); err != nil {
		log.Error("server failed", slog.String("error", err.Error()))
	}
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
	return log
}
