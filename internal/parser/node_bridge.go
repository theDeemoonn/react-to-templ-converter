package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"react-to-templ-converter/internal/models"
	"strconv"
	"sync"
	"time"
)

// NodeJSParser использует Node.js для парсинга React компонентов
type NodeJSParser struct {
	parserPath  string
	parserPort  int
	parserURL   string
	parserCmd   *exec.Cmd
	parserReady chan struct{}
	mutex       sync.Mutex
}

// NewNodeJSParser создает новый экземпляр парсера на Node.js
func NewNodeJSParser(parserPath string) *NodeJSParser {
	port := 3001 // порт по умолчанию

	// Проверяем переменную окружения для переопределения порта
	if envPort := os.Getenv("PARSER_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		}
	}

	return &NodeJSParser{
		parserPath:  parserPath,
		parserPort:  port,
		parserURL:   fmt.Sprintf("http://localhost:%d/parse", port),
		parserReady: make(chan struct{}),
	}
}

// StartParser запускает Node.js парсер как отдельный процесс
func (p *NodeJSParser) StartParser() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Проверяем, что парсер еще не запущен
	if p.parserCmd != nil && p.parserCmd.Process != nil {
		select {
		case <-p.parserReady:
			// Парсер уже готов
			return nil
		default:
			// Парсер запущен, но еще не готов
			select {
			case <-p.parserReady:
				return nil
			case <-time.After(5 * time.Second):
				// Таймаут, останавливаем существующий процесс
				p.StopParser()
			}
		}
	}

	// Находим node и npm в системе
	nodePath, err := exec.LookPath("node")
	if err != nil {
		return fmt.Errorf("не удалось найти node в системе: %w", err)
	}

	// Проверяем существование директории парсера
	if _, err := os.Stat(p.parserPath); os.IsNotExist(err) {
		return fmt.Errorf("директория парсера не существует: %s", p.parserPath)
	}

	// Проверяем, установлены ли зависимости
	nodeModulesPath := filepath.Join(p.parserPath, "node_modules")
	if _, err := os.Stat(nodeModulesPath); os.IsNotExist(err) {
		// Устанавливаем зависимости
		npmCmd := exec.Command("npm", "install")
		npmCmd.Dir = p.parserPath

		var npmOut bytes.Buffer
		npmCmd.Stdout = &npmOut
		npmCmd.Stderr = &npmOut

		if err := npmCmd.Run(); err != nil {
			return fmt.Errorf("ошибка установки зависимостей: %w\n%s", err, npmOut.String())
		}
	}

	// Запускаем парсер
	cmd := exec.Command(nodePath, "index.js")
	cmd.Dir = p.parserPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("PARSER_PORT=%d", p.parserPort))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ошибка запуска парсера: %w", err)
	}

	p.parserCmd = cmd

	// Ожидаем сигнала готовности от парсера
	go func() {
		scanner := bytes.NewBuffer(nil)
		for {
			n, err := stdout.WriteTo(scanner)
			if err != nil || n == 0 {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if bytes.Contains(scanner.Bytes(), []byte("PARSER_READY")) {
				close(p.parserReady)
				break
			}
		}
	}()

	// Ожидаем сигнала готовности с таймаутом
	select {
	case <-p.parserReady:
		log.Println("Парсер успешно запущен на порту", p.parserPort)
		return nil
	case <-time.After(10 * time.Second):
		p.StopParser()
		return fmt.Errorf("таймаут запуска парсера: %s", stderr.String())
	}
}

// StopParser останавливает процесс парсера
func (p *NodeJSParser) StopParser() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.parserCmd != nil && p.parserCmd.Process != nil {
		_ = p.parserCmd.Process.Kill()
		p.parserCmd = nil
		p.parserReady = make(chan struct{})
	}
}

// ParseComponent парсит React компонент и возвращает его структуру
func (p *NodeJSParser) ParseComponent(code string) (*models.ReactComponent, error) {
	// Запускаем парсер, если он еще не запущен
	if err := p.StartParser(); err != nil {
		return nil, fmt.Errorf("ошибка запуска парсера: %w", err)
	}

	// Готовим данные для отправки
	requestData := map[string]string{
		"code": code,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации данных: %w", err)
	}

	// Устанавливаем таймаут для запроса
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Создаем HTTP запрос
	req, err := http.NewRequestWithContext(ctx, "POST", p.parserURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Отправляем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// Пробуем перезапустить парсер и повторить запрос
		p.StopParser()
		if err := p.StartParser(); err != nil {
			return nil, fmt.Errorf("ошибка перезапуска парсера: %w", err)
		}

		// Создаем новый запрос с таймаутом
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, _ = http.NewRequestWithContext(ctx, "POST", p.parserURL, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("ошибка отправки запроса после перезапуска: %w", err)
		}
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ошибка ответа парсера (код %d): %s", resp.StatusCode, string(body))
	}

	// Парсим ответ
	var result models.ReactComponent
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	return &result, nil
}
