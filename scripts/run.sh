#!/bin/bash

cd terraform
terraform init
terraform plan  
run: terraform apply -var "access_key=${{ secrets.ACCESS_KEY }}" -var "secret_key=${{ secrets.SECRET_KEY }}" -auto-approve

HOST_IP=$(terraform output -raw ip_address)

ssh -o StrictHostKeyChecking=no -l maxim ${HOST_IP} "
  sudo apt-get install apt-transport-https ca-certificates curl gnupg lsb-release -y
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
  echo \"deb [arch=\$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \$(lsb_release -cs) stable\" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
  sudo apt-get update
  sudo apt-get install docker-ce docker-ce-cli containerd.io -y
  sudo apt-get install docker-compose-plugin -y
"
scp ../docker-compose.yml maxim@${HOST_IP}:/home/maxim/docker-compose.yml
#docker compose up -d
ssh -o StrictHostKeyChecking=no -l maxim ${HOST_IP} "
  cd /home/maxim
  docker compose up -d
  docker compose exec backend apt install golang-migrate
  docker compose exec backend migrate -path=./migrations -database "postgresql://validator:val1dat0r@postgres:5432/project-sem-1?sslmode=disable" -verbose up 
  docker compose exec backend  go run insertInDB/insert.go 
"
