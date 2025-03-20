.PHONY: setup build run test clean parser generate-templ install-deps

# Переменные
APP_NAME := react-to-templ-converter
BUILD_DIR := ./build
PARSER_DIR := ./parser-js
PORT := 8080
PARSER_PORT := 3001

# Настройка и установка зависимостей
setup: install-deps generate-templ

# Установка зависимостей Go и Node.js
install-deps:
	@echo "==> Установка зависимостей Go..."
	go mod tidy
	@echo "==> Установка зависимостей Node.js..."
	cd $(PARSER_DIR) && npm install

# Генерация шаблонов templ
generate-templ:
	@echo "==> Генерация шаблонов templ..."
	templ generate ./web/templates/

# Сборка приложения
build: generate-templ
	@echo "==> Сборка $(APP_NAME)..."
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server/

# Запуск сервера
run:
	@echo "==> Запуск сервера на порту $(PORT)..."
	PORT=$(PORT) PARSER_PORT=$(PARSER_PORT) go run ./cmd/server/

# Запуск тестов
test:
	@echo "==> Запуск тестов..."
	go test -v ./...

# Запуск только парсера
parser:
	@echo "==> Запуск парсера на порту $(PARSER_PORT)..."
	cd $(PARSER_DIR) && PARSER_PORT=$(PARSER_PORT) node index.js

# Очистка сборки
clean:
	@echo "==> Очистка..."
	rm -rf $(BUILD_DIR)
	find . -name "*_templ.go" -delete