package main

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"itmo-devops-sem1-project-template/internal/config"
	database "itmo-devops-sem1-project-template/internal/db"
	findcsv "itmo-devops-sem1-project-template/internal/findcsv"
	unpackage "itmo-devops-sem1-project-template/internal/tools"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	envLocal = "local"
)

var db *pgxpool.Pool

type InsertResponse struct {
	TotalCount      int `json:"total_count"`
	DuplicatesCount int `json:"duplicates_count"`
	TotalItems      int `json:"total_items"`
	TotalCategories int `json:"total_categories"`
	TotalPrice      int `json:"total_price"`
}

func main() {
	//подгрузка конфига
	cfg := config.MustLoad()
	log := setupLogger(cfg.Env)
	log.Info("got config", slog.String("env", cfg.Env))

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v0/prices", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			GetTheInfo(w, r)
		case http.MethodPost:
			UploadOnServer(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	loggedMux := LoggingMiddleware(log, mux)
	// получение данных из конфига
	srv := &http.Server{
		Addr:        cfg.Address,
		ReadTimeout: cfg.Timeout,
		IdleTimeout: cfg.IdleTimeout,
		Handler:     loggedMux,
	}

	//открытие соtдинения с бд
	var error error
	db, error = database.ConnectDB()
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

func LoggingMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Info("http request",
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
			slog.String("remote", r.RemoteAddr),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

// ручка на отправку файлов
func UploadOnServer(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	//считываем параметры из запроса
	fileType := r.URL.Query().Get("type")
	if fileType != "tar" && fileType != "zip" {
		http.Error(w, "Недоступный тип переданых данных", http.StatusInternalServerError)
		return
	}
	//ограничиваем размер файла 10mb

	maxarchsize := int64(10 << 20)
	r.Body = http.MaxBytesReader(w, r.Body, maxarchsize)

	//определяем тип передаваемого архива
	switch fileType {
	case "tar":
		err := unpackage.Untar(r.Body, "/tmp/extracted")
		if err != nil {
			http.Error(w, "Ошибка распаковки: "+err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Файл очень большой: "+err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		readerAt := bytes.NewReader(buf)
		size := int64(len(buf))
		err = unpackage.Unzip(readerAt, size, "/tmp/extracted")
		if err != nil {
			http.Error(w, "Ошибка распаковки: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// находим файл
	csvpath, error := findcsv.FindAnyCSV("/tmp/extracted")
	if error != nil {
		http.Error(w, "Ошибка поиска csv файла в архиве: "+error.Error(), http.StatusInternalServerError)
		return
	}
	if db == nil {
		http.Error(w, "Ошибка подключения к бд", http.StatusInternalServerError)
		return
	}

	f, err := os.Open(csvpath)
	if err != nil {
		http.Error(w, "Не удалось прочитать файл из архива(1): "+err.Error(), http.StatusInternalServerError)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		http.Error(w, "Не удалось прочитать файл из архива(2): "+err.Error(), http.StatusInternalServerError)
	}

	ctx := r.Context()
	var totalCount, totalItems, duplicatesCount int
	var totalPrice float64

	categories := make(map[string]struct{})
	tx, err := db.Begin(ctx)
	if err != nil {
		http.Error(w, "Ошибка создания транзакции: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// закрытие транзакции если что-то пойдет не так
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)

		} else if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	stmt := `
      INSERT INTO prices (name, category, price, create_date)
	  VALUES ($1, $2, $3, $4)
      ON CONFLICT (name, category, price, create_date) DO NOTHING
      RETURNING TRUE;
    `

	for _, rec := range records[1:] {
		totalCount++
		// проверка даты
		if _, err := time.Parse("2006-01-02", rec[4]); err != nil {
			continue
		}
		// проверка ценыф
		price, err := strconv.ParseFloat(rec[3], 64)
		if err != nil {
			continue
		}
		var inserted bool
		err = tx.QueryRow(ctx, stmt, rec[1], rec[2], price, rec[4]).Scan(&inserted)
		if err == sql.ErrNoRows {
			duplicatesCount++
		} else if err != nil {
			_ = tx.Rollback(ctx)
			http.Error(w, "Ошибка вставки значения: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if inserted {
			totalItems++
			categories[rec[2]] = struct{}{}
			totalPrice += price
		} else {
			duplicatesCount++
		}
	}
	// коммит транзакции
	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "Ошибка коммита транзакции:"+err.Error(), http.StatusInternalServerError)
		return
	}

	// удаление файла из временной директрии
	entries, _ := os.ReadDir("/tmp/extracted")
	for _, entry := range entries {
		_ = os.RemoveAll(filepath.Join("/tmp/extracted", entry.Name()))
	}

	// ответ
	res := InsertResponse{
		TotalCount:      totalCount,
		DuplicatesCount: duplicatesCount,
		TotalItems:      totalItems,
		TotalCategories: len(categories),
		TotalPrice:      int(totalPrice),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// ручка для получения файлов
func GetTheInfo(w http.ResponseWriter, r *http.Request) {

	// считываем параметры из запроса
	dateStart := r.URL.Query().Get("start")
	dateEnd := r.URL.Query().Get("end")
	min := r.URL.Query().Get("min")
	max := r.URL.Query().Get("max")

	// Валидация min
	var intMin, intMax int
	var err error
	if min != "" {
		intMin, err = strconv.Atoi(min)
		if err != nil {
			http.Error(w, "Неверный формат min", http.StatusBadRequest)
			return
		}
	}
	if max != "" {
		intMax, err = strconv.Atoi(max)
		if err != nil {
			http.Error(w, "Неверный формат max", http.StatusBadRequest)
			return
		}
	}
	// proverka logiki price
	if min != "" && max != "" && intMin > intMax {
		http.Error(w, "Минимальная цена не может быть больше максимальной", http.StatusBadRequest)
		return
	}

	// Валидируем dateStart, если он пришёл
	if dateStart != "" {
		if _, err := time.Parse("2006-01-02", dateStart); err != nil {
			http.Error(w, "Неверный формат начальной даты", http.StatusBadRequest)
			return
		}
	}
	// Валидируем dateEnd, если он пришёл
	if dateEnd != "" {
		if _, err := time.Parse("2006-01-02", dateEnd); err != nil {
			http.Error(w, "Неверный формат конечной даты", http.StatusBadRequest)
			return
		}
	}

	file, err := os.Create("/tmp/preextracted/data.csv")
	if err != nil {
		http.Error(w, "Ошибка создания файла csv: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	ctx := r.Context()

	rows, err := db.Query(ctx, `
    SELECT id, name, category, price, create_date
    FROM prices
    WHERE 
        ($1 IS NULL OR price >= $1)
        AND ($2 IS NULL OR price <= $2)
        AND ($3 IS NULL OR create_date >= $3)
        AND ($4 IS NULL OR create_date <= $4)
    `,
		intMin, intMax, dateStart, dateEnd)
	if err != nil {
		http.Error(w, "Не удалось считать данные из таблицы: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name, category string
		var price float64
		var create_date time.Time

		if err := rows.Scan(&id, &name, &category, &price, &create_date); err != nil {
			http.Error(w, "Ошибка при чтении данных из базы: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := writer.Write([]string{
			strconv.Itoa(id),
			name,
			category,
			fmt.Sprintf("%.2f", price),
			create_date.Format("2006-01-02"),
		}); err != nil {
			http.Error(w, "Ошибка записи CSV: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Ошибка при итерации по строкам: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		http.Error(w, "Ошибка при записи CSV: "+err.Error(), http.StatusInternalServerError)
		return
	}

	archive, err := os.Create("/tmp/preextracted/response.zip")
	if err != nil {
		http.Error(w, "Ошибка создания архива: "+err.Error(), http.StatusInternalServerError)
		return
	}

	zipWriter := zip.NewWriter(archive)

	f1, err := os.Open("/tmp/preextracted/data.csv")
	if err != nil {
		http.Error(w, "Ошибка открытия CSV файла для записи в архив: "+err.Error(), http.StatusInternalServerError)
		archive.Close()
		return
	}
	defer f1.Close()

	w1, err := zipWriter.Create("data.csv")
	if err != nil {
		http.Error(w, "Ошибка добавления файла в zip архив: "+err.Error(), http.StatusInternalServerError)
		f1.Close()
		archive.Close()
		return
	}

	if _, err := io.Copy(w1, f1); err != nil {
		http.Error(w, "Ошибка копирования файла в архив: "+err.Error(), http.StatusInternalServerError)
		f1.Close()
		archive.Close()
		return
	}

	if err := zipWriter.Close(); err != nil {
		http.Error(w, "Ошибка закрытия zip архива: "+err.Error(), http.StatusInternalServerError)
		archive.Close()
		return
	}

	if err := archive.Close(); err != nil {
		http.Error(w, "Ошибка закрытия файла архива: "+err.Error(), http.StatusInternalServerError)
		return
	}

	file, err = os.Open("/tmp/preextracted/response.zip")
	if err != nil {
		http.Error(w, "Ошибка получения архива", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "Ошибка получения информации об архиве", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileInfo.Name()))
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "Ошибка передачи архива: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// очистка директории
	entries, _ := os.ReadDir("/tmp/preextracted")
	for _, entry := range entries {
		_ = os.RemoveAll(filepath.Join("/tmp/preextracted", entry.Name()))
	}
}
