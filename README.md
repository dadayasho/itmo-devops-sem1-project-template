# Финальный проект 1 семестра

REST API сервис для загрузки и выгрузки данных о ценах.

## Требования к системе

- OS: Windows 11
- Processor(AMD): AMD Ryzen 7 7700X
- Processor(Intel): Intel Core i7-13700k
- Memory: 16GB
- Graphics Card(AMD): AMD Radeon RX 7700-XT
- Graphics Card(Nvidia): Nvidia RTX 4070Ti
- Graphics Card(Intel): Intel Arc B580
- Direct X: DX12
- Online Connection Requirements: Yes
- Hard Drive Space: SSD - 2GiB

---
## Установка и запуск

Чтобы запустить сервер следует выполнить несколько шагов.

1. Шаг - скопировать репозиторий
```
git clone git@github.com:dadayasho/itmo-devops-sem1-project-template.git
```
После копирования следует перейди в директорию проекта.
---
_Далее будут рассмотрены 2 сценария_
---
2.1 Шаг - локальный запуск

Следует понимать, что проект реализован под деплой на сервер, поэтому есть ряд особенностей для запуска сервера локально.

После копирования проекта следует собрать docker file командой:
```
docker build -t <your>/<tag> .
```
> [!IMPORTANT]
> Важно понимать, под какой архитектурой ОС вы работаете, но при локальном запуске не критично не указывать архитектуру
> Так же следует обратить внимание на `<your>/<tag>` - эта часть должна включать ваш удаленный репозиторий для хранения контейнеров.

Далее следует выполнить `push` в удаленный репозитриий, где вместо `<your>/<tag>` ваш удаленный репозиторий
```
docker push `<your>/<tag>`
```

Теперь переходим к предпоследнему шагу - создание файла с переменными окружения, которые будут переданы в облако.
Создаем `.env` файл в директории куда кладем следующие переменные
```
CONFIG_PATH = "<your_value>"
DB_USER_NAME = "<your_value>"
DB_USER_PASSWORD = "<your_value>"
DB_IP = "<your_value>"
DB_NAME = "<your_value>"
DB_PORT = "<your_value>"
```

Финальная часть шага - поднятие docker compose
В 20 строке `docker-compose.yml` вставить ваш удаленный репозиторий
```
docker compose up -d
```
Следует выполнить пару дополнительных шагов - миграции и заполнение таблицы `price данными`
```
sudo docker compose exec backend apt install golang-migrate
sudo docker compose exec backend migrate -path=./migrations -database "postgresql://validator:val1dat0r@postgres:5432/project-sem-1?sslmode=disable" -verbose up 
sudo docker compose exec backend  go run insertInDB/insert.go 
```
Готово, сервер работает локально, эндпоинты доступны по `http://localhost:8080`
---
2.2 Шаг - деплой на сервер.
---
Следует создать `.env` файл и поместить туда следущие переменные:
```
CONFIG_PATH = "<your_value>"
DB_USER_NAME = "<your_value>"
DB_USER_PASSWORD = "<your_value>"
DB_IP = "<your_value>"
DB_NAME = "<your_value>"
DB_PORT = "<your_value>"
AWS_ACCESS_KEY_ID = "<your_value>"
AWS_SECRET_ACCESS_KEY = "<your_value>"
YC_TOKEN = "<your_value>"
YC_CLOUD_ID = "<your_value>"
YC_FOLDER_ID = "<your_value>"
```

> Следует создать бакет в облаке, и дать права сервисному аккаунту на их редактирование. Получить статические клучи доступа.
> `AWS_ACCESS_KEY_ID` - токен от сервисного аккаунта
> `AWS_SECRET_ACCESS_KEY` - секретный ключ от сервисного аккаунта
> `YC_TOKEN` - токен для доступа в облако
> `YC_CLOUD_ID` - id облака
> `YC_FOLDER_ID` - id хранилища в облаке

Это все нужно для реализации хранения state файла состояния ресурсов в облаке в бакет, чтобы обращаться к нему в момент создания ресурсов во избежании коллизии.
---
Далее следует установить терраформ запустить скрипт `script/run.sh` чтобы собрать образ и оправить его на ваш удаленный репозиторий.
Указать ваш репозиторий в 20 строке файла `docker-compose.yml` 
После этого следует запустить скрипт `run.sh` - скрипт, который создает виртуальную машину со статическим ip адрессов в облаке.
Этот скрипт создает ВМ, устанавливает docker compose и поднимает сервер на порту `8080`, выполняет миграции и заполняет таблицу данными.

Готово, сервер поднят в облаке.
---
## Тестирование

Директория `sample_data` - это пример директории, которая является разархивированной версией файла `sample_data.zip`

Сервер был протестирован 2 разными тестами.

1. Через Postman
---
![Проверка ручки POST - успешная](pictures/PostConfirm.png)
---
![Проверка ручки GET - успешная](pictures/GetConfirm.png)
На сервер передан архив, который был успешно обработан, содержимое вставлено в таблицу
---
![Проверка ручки GET - провал](pictures/GetWrong.png)
Отправка запросов с невалидными параметрами
---
![Проверка ручки POST - провал](pictures/PostWrong.png)
Передача неверного параметра не равного "zip" или "tar" в параметрах запроса.
---
2. Автоматизироанная проверка через GitHub actions с помощью скрипта `scripts/tests.sh`
Перед проверкой средует создать переменную окружения `HOST_IP` с ip адрессом сервера.

![Проверка через скрипт - успешна](pictures/ScriptTests.png)

## Контакт

Останусь инкогнито
