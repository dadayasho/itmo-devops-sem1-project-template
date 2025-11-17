FROM golang:alpine AS builder 

RUN mkdir -p /tmp/preextracted
RUN mkdir -p /tmp/extracted

WORKDIR /itmo-devops-sem1-project-template


COPY . .
RUN go mod download
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

RUN go build -o app ./...

EXPOSE 8080

CMD ["./app"]

