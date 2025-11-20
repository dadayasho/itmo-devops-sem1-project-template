#!/bin/bash
#утилка чтоб парсить json, так как я не придумал ничего лучше для получения IP с уже готовой виртуальной машины
echo "======= Утсанавливаем jq ========"
sudo apt-get install jq -y

echo "======= Устанавливаем yc ======="
curl https://storage.yandexcloud.net/yandexcloud-yc/install.sh | bash -s -- -a
export PATH="$HOME/yandex-cloud/bin:$PATH"

echo "=======Авторизуемся в yc======="

yc config set token ${YC_TOKEN}
echo "======= Добавляем переменные окружения в .env ======="

echo "POSTGRES_USER=${POSTGRES_USER}" > .env
echo "POSTGRES_PASSWORD=${POSTGRES_PASSWORD}" >> .env
echo "POSTGRES_DB=${POSTGRES_DB}" >> .env
echo "POSTGRES_PORT=${POSTGRES_PORT}" >> .env
echo "POSTGRES_HOST=${POSTGRES_HOST}" >> .env
echo 'CONFIG_PATH=/itmo-devops-sem1-project-template/config/local.yaml' >> .env

echo "======= Добавляем SSH ключик ======"
mkdir -p ~/.ssh
echo "$SSH_PRIVATE_KEY" > ~/.ssh/id_rsa
chmod 600 ~/.ssh/id_rsa
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_rsa
echo "$SSH_PUBLIC_KEY" > ~/.ssh/id_rsa.pub
chmod 644 ~/.ssh/id_rsa.pub

echo "======= Создаем виртуалку ========"

get_vm_info() {
  local vm_info_json
  vm_info_json=$(yc compute instance list --folder-id "$YC_FOLDER_ID" --format json)

  local vm_id=$(echo "$vm_info_json" | jq -r --arg vmname go-server-vm '.[] | select(.name == $vmname) | .id')
  if [ -z "$vm_id" ]; then
    echo >&2 "ВМ с именем go-server-vm не найдена"
    return
  fi
  echo "$vm_id"
  
}

get_ip() {
  local vm_info_json
  vm_info_json=$(yc compute instance list --folder-id "$YC_FOLDER_ID" --format json)
    local external_ip=$(echo "$vm_info_json" | jq -r --arg vmname go-server-vm '
    .[] | select(.name == $vmname) | .network_interfaces[0].primary_v4_address.one_to_one_nat.address // empty
  ')

  echo "$external_ip"
}

CREATED=1
CREATE_OUTPUT=$(yc compute instance create \
  --cloud-id "$YC_CLOUD_ID" \
  --folder-id "$YC_FOLDER_ID" \
  --zone ru-central1-a \
  --name go-server-vm \
  --platform standard-v3 --cores 2 --memory 2 \
  --network-interface subnet-id="$YC_SUBNET_ID" \
  --create-boot-disk image-folder-id=standard-images,image-family=ubuntu-2204-lts,size=20 \
  --ssh-key ~/.ssh/id_rsa.pub --format yaml 2>&1) || CREATED=0

if [ "$CREATED" = "0" ]; then
  if echo "$CREATE_OUTPUT" | grep -q 'Instance with name.*already exists'; then
    echo "VM уже существует, получаем IP..."
    read VM_ID <<< "$(get_vm_info)"
    read HOST_IP <<< "$(get_ip)"
    
  else
    echo "Ошибка создания VM:"
    echo "$CREATE_OUTPUT"
    exit 1
  fi
else
  VM_ID=$(echo "$CREATE_OUTPUT" | grep '^id:' | sed 's/id: //')
  echo "VM создана с ID=$VM_ID"
  HOST_IP=$(yc vpc address create --folder-id $YC_FOLDER_ID --name static-adr --external-ipv4 zone=ru-central1-a --format yaml | grep 'address:' | sed -n 's/^[[:space:]]*address:[[:space:]]*//p')
  echo "IP создан: $HOST_IP"
  yc compute instance add-one-to-one-nat --id="$VM_ID" --network-interface-index=0 --nat-address="$HOST_IP"
fi

echo "VM_ID: $VM_ID"
echo "HOST_IP: $HOST_IP"


echo "====== ПОДКЛЮЧАЕМСЯ ПО SHH ======"

sleep 80

ssh-keyscan -H "$HOST_IP" >> ~/.ssh/known_hosts

echo "====== Устанавливаем DOCKER ======"
ssh -o StrictHostKeyChecking=no -l yc-user $HOST_IP "
  sudo apt-get install apt-transport-https ca-certificates curl gnupg lsb-release -y
  sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
  sudo echo \"deb [arch=\$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \$(lsb_release -cs) stable\" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
  sudo apt-get update
  sudo apt-get install docker-ce docker-ce-cli containerd.io -y
  sudo apt-get install docker-compose-plugin -y
"
echo "====== Копируем переменные на сервер ======="
scp .env yc-user@$HOST_IP:/home/yc-user/.env

echo "====== Docker compose на сервер ======"
scp docker-compose.yml yc-user@$HOST_IP:/home/yc-user/docker-compose.yml

echo "====== Поднимаем бэкэнд ======"
ssh -o StrictHostKeyChecking=no -l yc-user $HOST_IP "
  cd /home/yc-user
  sudo docker compose down
  sleep 5
  sudo docker rmi tonysoprano228/go-server:latest
  sudo docker compose up -d
  
  sleep 15
  echo "======= Выполняем миграции ======= "
  sudo docker compose exec backend migrate -path=./migrations -database "postgresql://validator:val1dat0r@postgres:5432/project-sem-1?sslmode=disable" -verbose up 
  sleep 15
  echo "======= Вставляем значения ======= "
  sudo docker compose exec backend  go run insertInDB/insert.go 
"

echo "Передача IP"
echo "HOST_IP=$HOST_IP" >> $GITHUB_OUTPUT
echo $HOST_IP