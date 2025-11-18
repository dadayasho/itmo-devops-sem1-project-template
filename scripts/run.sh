#!/bin/bash

cd terraform
env | grep AWS
env | grep YC
env | grep DB
env | grep SSH
cat > terraform.tfvars <<EOF
token     = "${YC_TOKEN}"
cloud_id  = "${YC_CLOUD_ID}"
folder_id = "${YC_FOLDER_ID}"
EOF
terraform init -reconfigure
terraform plan  
terraform apply -auto-approve

HOST_IP=$(terraform output -raw ip_address)

# ====== ПОДКЛЮЧАЕМСЯ ПО SHH ======
mkdir -p ~/.ssh
echo "$SSH_PRIVATE_KEY" > ~/.ssh/id_rsa
chmod 600 ~/.ssh/id_rsa

eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_rsa


ssh-keyscan -H "${HOST_IP}" >> ~/.ssh/known_hosts

#====== Устанавливае DOCER ======
ssh -o StrictHostKeyChecking=no -l maxim ${HOST_IP} "
  sudo apt-get install apt-transport-https ca-certificates curl gnupg lsb-release -y
  sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
  sudo echo \"deb [arch=\$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \$(lsb_release -cs) stable\" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
  sudo apt-get update
  sudo apt-get install docker-ce docker-ce-cli containerd.io -y
  sudo apt-get install docker-compose-plugin -y
"
# ====== Копируем переменные на сервер =======
scp ../.env maxim@${HOST_IP}:/home/maxim/.env

# ====== Docker compose на сервер ======
scp ../docker-compose.yml maxim@${HOST_IP}:/home/maxim/docker-compose.yml

# ====== Поднимаем бэкэнд ======
ssh -o StrictHostKeyChecking=no -l maxim ${HOST_IP} "
  cd /home/maxim
  sudo docker compose down
  sudo docker compose up -d
  sudo docker compose exec backend apt install golang-migrate
  sudo docker compose exec backend migrate -path=./migrations -database "postgresql://validator:val1dat0r@postgres:5432/project-sem-1?sslmode=disable" -verbose up 
  sudo docker compose exec backend  go run insertInDB/insert.go 
"
