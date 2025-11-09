package main

import (
	//"fmt"
	"log/slog"
	"os"

	"itmo-devops-sem1-project-template/internal/config"
	"itmo-devops-sem1-project-template/internal/db"
)

const (
	envLocal = "local"
)

func main() {
	cfg := config.MustLoad()
	//fmt.Println(cfg)
	log := setupLogger(cfg.Env)
	log.Info("starting server", slog.String("env", cfg.Env))

	db, error := db.СonnectDB()
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
