FROM golang:alpine AS builder 

RUN mkdir -p /tmp/preextracted
RUN mkdir -p /tmp/extracted

WORKDIR /itmo-devops-sem1-project-template

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY internal ./internal
COPY config ./config
COPY migrations ./migrations
COPY sample_data ./sample_data
COPY insertInDB ./insertInDB
COPY .env ./

RUN go build -o itmo-devops-sem1-project-template .
EXPOSE 8080

CMD ["./itmo-devops-sem1-project-template"]

