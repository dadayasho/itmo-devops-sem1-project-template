FROM golang:1.24.0 AS builder 

RUN mkdir -p /tmp/preextracted
RUN mkdir -p /tmp/extracted

WORKDIR /itmo-devops-sem1-project-template

COPY main.go ./
COPY internal ./internal
COPY config ./config
COPY migrations ./migrations
COPY sample_data ./sample_data
COPY insertInDB ./insertInDB
COPY go.mod go.sum ./
RUN go mod download
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest


RUN go build -o itmo-devops-sem1-project-template .
EXPOSE 8080

CMD ["./itmo-devops-sem1-project-template"]