# Многоэтапная сборка для оптимизации размера образа

# Этап 1: Сборка templ
FROM golang:1.24.1-alpine AS templ-builder
WORKDIR /build

# Установка Git и необходимых зависимостей
RUN apk add --no-cache git build-base

# Установка templ
RUN go install github.com/a-h/templ/cmd/templ@latest

# Этап 2: Сборка Node.js парсера
FROM node:20-alpine AS node-builder
WORKDIR /parser

# Копирование package.json и установка зависимостей
COPY parser-js/package*.json ./
RUN npm install

# Копирование кода парсера
COPY parser-js/ ./

# Этап 3: Сборка Go приложения
FROM golang:1.24.1-alpine AS go-builder
WORKDIR /app

# Установка Git и необходимых зависимостей
RUN apk add --no-cache git build-base

# Копирование go.mod и go.sum
COPY go.mod go.sum ./
RUN go mod download

# Копирование исходного кода
COPY . .

# Копирование templ из первого этапа
COPY --from=templ-builder /go/bin/templ /go/bin/templ

# Генерация templ шаблонов
RUN templ generate ./web/templates/

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux go build -o /react-to-templ-converter ./cmd/server/

# Этап 4: Финальный образ
FROM alpine:3.18

# Установка Node.js, npm и другие зависимости
RUN apk add --no-cache nodejs npm curl tzdata ca-certificates

# Создание рабочей директории
WORKDIR /app

# Копирование Node.js парсера
COPY --from=node-builder /parser /app/parser-js

# Копирование скомпилированного Go приложения
COPY --from=go-builder /react-to-templ-converter .

# Копирование статических файлов и примеров
COPY web/static/ ./web/static/
#COPY examples/ ./examples/

# Создание каталога для сгенерированных шаблонов
RUN mkdir -p ./web/templates

# Копирование сгенерированных templ шаблонов
COPY --from=go-builder /app/web/templates/*.go ./web/templates/

# Настройка окружения
ENV PORT=8080
ENV PARSER_PORT=3001
ENV TZ=Europe/Moscow

# Проверка здоровья контейнера
HEALTHCHECK --interval=30s --timeout=3s \
  CMD curl -f http://localhost:$PORT || exit 1

# Открытие порта
EXPOSE $PORT

# Запуск приложения
CMD ["./react-to-templ-converter"]