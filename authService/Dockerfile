FROM golang:1.24-alpine AS builder

RUN echo "http://dl-cdn.alpinelinux.org/alpine/v3.23/main" > /etc/apk/repositories && \
    echo "http://dl-cdn.alpinelinux.org/alpine/v3.23/community" >> /etc/apk/repositories

RUN apk update && apk add --no-cache \
    build-base \
    sqlite-dev

RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

RUN go install github.com/go-task/task/v3/cmd/task@latest

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /app/myapp ./cmd/sso/main.go

# Финальный образ
FROM alpine:latest

RUN echo "http://dl-cdn.alpinelinux.org/alpine/v3.23/main" > /etc/apk/repositories && \
    echo "http://dl-cdn.alpinelinux.org/alpine/v3.23/community" >> /etc/apk/repositories

RUN apk update && apk add --no-cache sqlite-libs
RUN apk add --no-cache postgresql-client ca-certificates

COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /go/bin/task /usr/local/bin/task
COPY --from=builder /app/myapp /usr/local/bin/myapp

WORKDIR /app

COPY Taskfile.yaml ./
COPY migrations/ ./migrations/
COPY config/ ./config/

ENV MAIN_DB_URL="postgres://user:pass@localhost:5432/main?sslmode=disable"
ENV SESSION_DB_URL="postgres://user:pass@localhost:5431/sessions?sslmode=disable"
ENV MAIN_MIGRATIONS_PATH="./migrations/main/postgres"
ENV SESSIONS_MIGRATIONS_PATH="./migrations/sessions/postgres"
ENV CONFIG_PATH="./config/local.yaml"

EXPOSE 50000

CMD ["myapp"]

