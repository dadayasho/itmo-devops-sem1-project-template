#!/bin/bash



apt install golang-migrate # добавить migrate в 
migrate -path=./migrations \
  -database "postgresql://validator:val1dat0r@localhost:5432/project-sem-1?sslmode=disable" \
  -verbose up # миграции бд
go run internal/insertInDB/insert.go # вставка данных в таблицу

