package main

import (
	"encoding/json"
	"fmt"
	"github.com/a-h/templ"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"react-to-templ-converter/internal/config"
	"react-to-templ-converter/internal/converter"
	"react-to-templ-converter/internal/generator"
	"react-to-templ-converter/internal/parser"
	"react-to-templ-converter/web/templates"
	"strings"
	"syscall"
	"time"
)

func main() {
	// Настройка логов
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Запуск конвертера React -> Templ + HTMX")

	// Получение порта из переменной окружения или использование значения по умолчанию
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Создаем экземпляр парсера
	reactParser := parser.NewNodeJSParser("./parser-js")

	// Запускаем парсер если он требует запуска
	if err := reactParser.StartParser(); err != nil {
		log.Fatalf("Ошибка запуска парсера: %v", err)
	}
	defer reactParser.StopParser()

	// Создаем маршрутизатор
	r := mux.NewRouter()

	// Статические файлы
	fs := http.FileServer(http.Dir("./web/static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// Примеры
	examplesFs := http.FileServer(http.Dir("./examples"))
	r.PathPrefix("/examples/").Handler(http.StripPrefix("/examples/", examplesFs))

	// Главная страница
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		component := templates.IndexPage()
		templ.Handler(component).ServeHTTP(w, r)
	}).Methods("GET")

	// API для конвертации
	r.HandleFunc("/api/convert", handleConversion(reactParser)).Methods("POST")

	// API для получения примеров
	r.HandleFunc("/api/examples/{name}", handleGetExample).Methods("GET")

	// Запуск сервера
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запуск сервера в горутине
	go func() {
		log.Printf("Сервер запущен на порту %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	// Ожидание сигнала для грациозного завершения
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	log.Println("Завершение работы сервера...")

	// Завершение работы сервера
	srv.Shutdown(nil)
	log.Println("Сервер остановлен")
}

// handleConversion обрабатывает запрос на конвертацию
func handleConversion(reactParser parser.ReactParser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка размера файла
		r.Body = http.MaxBytesReader(w, r.Body, 5<<20) // 5 МБ

		// Парсинг формы
		if err := r.ParseMultipartForm(5 << 20); err != nil {
			http.Error(w, "Ошибка разбора формы: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Получение файла
		file, header, err := r.FormFile("reactComponent")
		if err != nil {
			http.Error(w, "Ошибка получения файла: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Проверка расширения файла
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext != ".tsx" && ext != ".jsx" && ext != ".js" && ext != ".ts" {
			http.Error(w, "Неподдерживаемый формат файла. Разрешены только: .tsx, .jsx, .js, .ts", http.StatusBadRequest)
			return
		}

		// Чтение содержимого файла
		content, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Ошибка чтения файла: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Получение параметров конвертации
		//useHtmx := r.FormValue("useHtmx") == "true"
		componentName := strings.TrimSuffix(header.Filename, ext)

		// Создание опций для конвертера
		options := config.NewDefaultOptions()
		//options.UseHtmx = useHtmx
		options.UseHtmx = true
		options.ComponentName = componentName
		options.PackageName = "templates"
		options.Debug = false

		// Создаем все необходимые компоненты системы конвертации
		// JSX конвертер
		jsxConverter := converter.NewJSXToHTMXConverter(options)

		// Обработчик состояний
		stateHandler := converter.NewStateHandler(options)

		// Генератор templ шаблонов
		templGenerator := generator.NewTemplGenerator(options)
		templGenerator.SetJSXConverter(jsxConverter)

		// Генератор Go контроллеров
		goGenerator := generator.NewGoGenerator(options)
		goGenerator.SetStateHandler(stateHandler)

		// Создаем экземпляр конвертера
		reactConverter := converter.NewConverter(reactParser,
			converter.WithDebugMode(options.Debug),
			converter.WithIndentation(options.Indentation.Style, options.Indentation.Size))

		// Устанавливаем генераторы для конвертера
		if converterImpl, ok := reactConverter.(*converter.ReactToTemplConverter); ok {
			converterImpl.SetTemplGenerator(templGenerator)
			converterImpl.SetGoGenerator(goGenerator)
		}

		// Конвертация
		result, err := reactConverter.Convert(string(content), options)
		if err != nil {
			http.Error(w, "Ошибка конвертации: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Создаем ответ
		response := map[string]interface{}{
			"templFile":    result.TemplFile,
			"goController": result.GoController,
			"htmxJS":       result.HtmxJS,
		}

		// Возвращаем результат
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Ошибка сериализации ответа: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Логируем успешную конвертацию
		log.Printf("Успешно сконвертирован компонент %s", componentName)
	}
}

// handleGetExample обрабатывает запрос на получение примера
func handleGetExample(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	// Проверка безопасности имени файла
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		http.Error(w, "Некорректное имя файла", http.StatusBadRequest)
		return
	}

	// Путь к файлу примера
	filePath := fmt.Sprintf("./examples/react/%s.tsx", name)

	// Проверка существования файла
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Пример не найден", http.StatusNotFound)
		return
	}

	// Чтение содержимого файла
	content, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "Ошибка чтения файла: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращаем содержимое примера
	w.Header().Set("Content-Type", "text/plain")
	w.Write(content)
}
