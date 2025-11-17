#!/bin/bash

terraform plan -var-file=var.tfvars
#работа терраформа
HOST_IP = $(terraform output -raw ip_address)

scp docker-compose.yml maxim@$(HOST_IP):/
ssh -l maxim $(HOST_IP)
docker compose up -d

docker compose exec go-server apt install golang-migrate # добавить migrate в 
docker compose exec go-server migrate -path=./migrations -database "postgresql://validator:val1dat0r@postgres:5432/project-sem-1?sslmode=disable" -verbose up # миграции бд
docker compose exec go-server  go run insertInDB/insert.go # вставка данных в таблицу

