#!/bin/bash

echo "======= Устанавливаем yc ======="
curl https://storage.yandexcloud.net/yandexcloud-yc/install.sh | bash -s -- -a


echo "======= Добавляем переменные окружения в .env ======="

echo "POSTGRES_USER=${{ secrets.DB_USER_NAME }}" > .env
echo "POSTGRES_PASSWORD=${{ secrets.DB_USER_PASSWORD }}" >> .env
echo "POSTGRES_DB=${{ secrets.DB_NAME }}" >> .env
echo "POSTGRES_PORT=${{ secrets.DB_PORT }}" >> .env
echo "POSTGRES_HOST=${{ secrets.DB_IP }}" >> .env
echo 'CONFIG_PATH=/itmo-devops-sem1-project-template/config/local.yaml' >> .env

echo "======= Добавляем SSH ключик ======"

echo "$SSH_PUBLIC_KEY" >> ~/.ssh/id_rsa.pub

echo "======= Создаем виртуалку ========"

HOST_ID=$(yc compute instance create \
  --cloud-id b1gtthkbpv7pj92rkj82 \
  --folder-id b1grdmsmf3ucpm9annq2 \
  --zone ru-central1-a \
  --name go-server-vm \
  --platform standard-v3 \
  --cores 2 \
  --memory 2  \
  --network-interface subnet-id=e9bgulek7pok5ebk6qls \
  --create-boot-disk image-folder-id=standard-images,image-family=ubuntu-2204-lts,size=20 \
  --ssh-key ~/.ssh/id_ed25519.pub --format yaml | grep '^id:' | sed 's/id: //')


HOST_IP=$(yc vpc address create --name my-address --external-ipv4 zone=ru-central1-a --format yaml | grep 'address:' | sed -n 's/^[[:space:]]*address: //p')


yc compute instance add-one-to-one-nat \
  --id=${HOST_ID} \
  --network-interface-index=0\
  --nat-address=${HOST_IP}

echo "====== ПОДКЛЮЧАЕМСЯ ПО SHH ======"
mkdir -p ~/.ssh
echo "$SSH_PRIVATE_KEY" > ~/.ssh/id_rsa
chmod 600 ~/.ssh/id_rsa

eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_rsa


ssh-keyscan -H "${HOST_IP}" >> ~/.ssh/known_hosts

echo "====== Устанавливае DOCKER ======"
ssh -o StrictHostKeyChecking=no -l yc-user ${HOST_IP} "
  sudo apt-get install apt-transport-https ca-certificates curl gnupg lsb-release -y
  sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
  sudo echo \"deb [arch=\$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \$(lsb_release -cs) stable\" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
  sudo apt-get update
  sudo apt-get install docker-ce docker-ce-cli containerd.io -y
  sudo apt-get install docker-compose-plugin -y
"
ecgo "====== Копируем переменные на сервер ======="
scp ../.env yc-user@${HOST_IP}:/home/maxim/.env

echo "====== Docker compose на сервер ======"
scp ../docker-compose.yml yc-user@${HOST_IP}:/home/maxim/docker-compose.yml

echo "====== Поднимаем бэкэнд ======"
ssh -o StrictHostKeyChecking=no -l uc-user ${HOST_IP} "
  cd /home/maxim
  sudo docker compose down
  sudo docker compose up -d
  sudo docker compose exec backend apt install golang-migrate
  sudo docker compose exec backend migrate -path=./migrations -database "postgresql://validator:val1dat0r@postgres:5432/project-sem-1?sslmode=disable" -verbose up 
  sudo docker compose exec backend  go run insertInDB/insert.go 
"
