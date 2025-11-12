package main

import (
	"archive/zip"
	"bytes"
	"context"
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
)

const (
	envLocal = "local"
)

type InsertResponse struct {
	TotalCount      int `yaml:"total_count"`
	DuplicatesCount int `yaml:"duplicates_count"`
	TotalItems      int `yaml:"total_items"`
	TotalCategories int `yaml:"total_categories"`
	TotalPrice      int `yaml:"total_price"`
}

func main() {
	//подгрузка конфига
	cfg := config.MustLoad()
	log := setupLogger(cfg.Env)
	log.Info("got config", slog.String("env", cfg.Env))

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v0/prices", UploadOnServer)
	mux.HandleFunc("/api/v0/prices", GetTheInfo)

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

// ручка на отправку файлов
func UploadOnServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	//считываем параметры из запроса
	fileType := r.URL.Query().Get("type")

	//определяем тип передаваемого архива
	switch fileType {
	case "tar":
		err := unpackage.Untar(r.Body, "./tmp/extracted")
		if err != nil {
			http.Error(w, "Ошибка распаковки: "+err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body: "+err.Error(), http.StatusInternalServerError)
			return
		}
		readerAt := bytes.NewReader(buf)
		size := int64(len(buf))
		err = unpackage.Unzip(readerAt, size, "./tmp/extracted")
		if err != nil {
			http.Error(w, "Ошибка распаковки: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// находим файл
	csvpath, error := findcsv.FindAnyCSV("./tmp/extracted")
	if error != nil {
		http.Error(w, "Ошибка поиска csv файла в архиве: "+error.Error(), http.StatusInternalServerError)
		return
	}

	// подключаемся к бд
	db, error := database.СonnectDB()
	if error != nil {
		http.Error(w, "Не удалось подключиться к базе данных: "+error.Error(), http.StatusInternalServerError)
	}
	defer db.Close()

	// вставка в cсожержимого архива в бд с подсчетом по условию
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

	ctx := context.Background()

	// переопределяем счетчики
	total_count := 0
	duplicates_count := 0
	total_items := 0
	total_categories := 0

	// логика набивания данных
	var inserted bool
	categories := make(map[string]struct{})
	for _, rec := range records {
		if _, err1 := time.Parse("2006-01-02", rec[4]); err1 == nil {
			if _, err2 := strconv.ParseFloat(rec[3], 64); err2 == nil {
				err := db.QueryRow(ctx, `
					INSERT INTO prices (id, name, category, price, create_date)
					VALUES ($1, $2, $3, $4, $5)
					ON CONFLICT (id) DO UPDATE SET
						name = EXCLUDED.name,
						category = EXCLUDED.category,
						price = EXCLUDED.price,
						create_date = EXCLUDED.create_date
					RETURNING (xmax = 0) AS inserted;
				`, rec[0], rec[1], rec[2], rec[3], rec[4]).Scan(&inserted)
				if err != nil {
				}
				if inserted {
					total_items++
					if _, exists := categories[rec[1]]; !exists {
						categories[rec[1]] = struct{}{}
						total_categories++
					}
				} else {
					duplicates_count++
				}
				total_count++
			} else {
				total_count++
			}
		} else {
			total_count++
		}

	}

	// подсчет тотальной цены
	var total_price int
	_ = db.QueryRow(ctx, "SELECT SUM(price) FROM prices").Scan(&total_price)

	// очистка директории с архивом
	entries, _ := os.ReadDir("./tmp/extracted")
	for _, entry := range entries {
		_ = os.RemoveAll(filepath.Join("./tmp/extracted", entry.Name()))
	}

	// создание ответа
	body, _ := json.Marshal(&InsertResponse{
		TotalCount:      total_count,
		DuplicatesCount: duplicates_count,
		TotalItems:      total_items,
		TotalCategories: total_categories,
		TotalPrice:      total_price,
	})

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

// ручка для получения файлов
func GetTheInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method allowed", http.StatusMethodNotAllowed)
		return
	}

	//считываем параметры из запроса
	dateStart := r.URL.Query().Get("start")
	dateEnd := r.URL.Query().Get("end")
	min := r.URL.Query().Get("min")
	max := r.URL.Query().Get("max")

	// валидация параметров
	int_min, _ := strconv.Atoi(min)
	int_max, _ := strconv.Atoi(max)

	// проверка передаваемых значений
	if int_min <= 0 && int_max <= 0 {
		http.Error(w, "Неверный тип передаваемого значения цены", http.StatusMethodNotAllowed)
	}
	if _, err := time.Parse("2006-01-02", dateStart); err != nil {
		http.Error(w, "Неверный тип передаваемого значения начальной даты", http.StatusMethodNotAllowed)
	}
	if _, err := time.Parse("2006-01-02", dateEnd); err != nil {
		http.Error(w, "Неверный тип передаваемого значения конечной даты", http.StatusMethodNotAllowed)
	}

	// подключение к бд
	db, error := database.СonnectDB()
	if error != nil {
		http.Error(w, "Не удалось подключиться к базе данных: "+error.Error(), http.StatusInternalServerError)
	}
	defer db.Close()

	// создание файла csv
	file, err := os.Create("./tmp/preextracted/output.csv")
	if err != nil {
		http.Error(w, "Ошибка создания файла csv: "+error.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Write([]string{"id", "name", "category", "price", "create_date"})
	defer file.Close()
	defer writer.Flush()

	// получение данных с таблицы
	ctx := context.Background()
	rows, error := db.Query(ctx, `
    SELECT id, name, category, price, create_date FROM prices
    WHERE price > $1 AND price < $2 AND create_date > $3 AND create_date < $4
	`, int_min, int_max, dateStart, dateEnd)
	if error != nil {
		http.Error(w, "Не удалось подключиться считать данные из из таблицы: "+error.Error(), http.StatusInternalServerError)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		var category string
		var price float64
		var create_date string
		_ = rows.Scan(&id, &name, &category, &price, &create_date)

		writer.Write([]string{
			strconv.Itoa(id),
			name,
			category,
			fmt.Sprintf("%.2f", price),
			create_date,
		})
	}

	// создание архива
	archive, err := os.Create("./tmp/preextracted/archive.zip")
	if err != nil {
		http.Error(w, "Ошибка создания архива: "+error.Error(), http.StatusInternalServerError)
	}
	defer archive.Close()
	zipWriter := zip.NewWriter(archive)

	// добавление файла в zip
	f1, err := os.Open("./tmp/preextracted/output.csv")
	if err != nil {
		http.Error(w, "Ошибка открытия архива для записи"+error.Error(), http.StatusInternalServerError)
	}
	defer f1.Close()
	w1, err := zipWriter.Create("csv/output.csv")
	if err != nil {
		http.Error(w, "Ошибка добавления файла в zip архив:"+error.Error(), http.StatusInternalServerError)
	}
	if _, err := io.Copy(w1, f1); err != nil {
		panic(err)
	}
	zipWriter.Close()

	// Открытие файла
	file, err = os.Open("./tmp/preextracted/archive.zip")
	if err != nil {
		http.Error(w, "Ошипка получения архива", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Получение информации о файле
	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "Ошибка получения информации об архиве", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+fileInfo.Name()+`"`)
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// очистка репоризотория
	entries, _ := os.ReadDir("./tmp/preextracted")
	for _, entry := range entries {
		_ = os.RemoveAll(filepath.Join("./tmp/preextracted", entry.Name()))
	}
}
