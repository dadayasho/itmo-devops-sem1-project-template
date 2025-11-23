FROM golang:1.24.0 AS builder

WORKDIR /itmo-devops-sem1-project-template

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o itmo-devops-sem1-project-template .

FROM alpine:latest AS runner

WORKDIR /itmo-devops-sem1-project-template

COPY config ./config
COPY insertInDB ./insertInDB
COPY migrations ./migrations
COPY --from=builder /itmo-devops-sem1-project-template/itmo-devops-sem1-project-template ./
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate

EXPOSE 8080
CMD ["./itmo-devops-sem1-project-template"]
