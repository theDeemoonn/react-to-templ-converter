package models

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ConversionResult содержит результат конвертации React в Templ
type ConversionResult struct {
	TemplFile    string `json:"templFile"`    // Templ шаблон
	GoController string `json:"goController"` // Go контроллер
	HtmxJS       string `json:"htmxJS"`       // JavaScript для HTMX
	PropsStruct  string `json:"propsStruct"`  // Структура Go для пропсов

	ComponentName string                 `json:"componentName"` // Имя компонента
	SourceFile    string                 `json:"sourceFile"`    // Имя исходного файла
	ConvertedAt   string                 `json:"convertedAt"`   // Время конвертации
	Settings      map[string]interface{} `json:"settings"`      // Настройки конвертации
}

// NewConversionResult создает новый результат конвертации
func NewConversionResult(componentName string, sourceFile string) *ConversionResult {
	return &ConversionResult{
		ComponentName: componentName,
		SourceFile:    sourceFile,
		ConvertedAt:   time.Now().Format(time.RFC3339),
		Settings:      make(map[string]interface{}),
	}
}

// SaveToFiles сохраняет результаты конвертации в файлы
func (r *ConversionResult) SaveToFiles(outputDir string) error {
	// Создаем директорию если не существует
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории: %w", err)
	}

	// Сохраняем templ файл
	if r.TemplFile != "" {
		templPath := filepath.Join(outputDir, r.getTemplFileName())
		if err := os.WriteFile(templPath, []byte(r.TemplFile), 0644); err != nil {
			return fmt.Errorf("ошибка записи templ файла: %w", err)
		}
	}

	// Сохраняем Go контроллер
	if r.GoController != "" {
		goPath := filepath.Join(outputDir, r.getGoFileName())
		if err := os.WriteFile(goPath, []byte(r.GoController), 0644); err != nil {
			return fmt.Errorf("ошибка записи Go файла: %w", err)
		}
	}

	// Сохраняем JavaScript для HTMX
	if r.HtmxJS != "" {
		jsPath := filepath.Join(outputDir, r.getJSFileName())
		if err := os.WriteFile(jsPath, []byte(r.HtmxJS), 0644); err != nil {
			return fmt.Errorf("ошибка записи JavaScript файла: %w", err)
		}
	}

	return nil
}

// SaveToZip создает ZIP-архив с результатами конвертации
func (r *ConversionResult) SaveToZip(outputPath string) error {
	// В реальном приложении здесь будет код для создания ZIP-архива
	// Для простоты этот метод оставлен без реализации
	return nil
}

// getTemplFileName возвращает имя templ файла
func (r *ConversionResult) getTemplFileName() string {
	return fmt.Sprintf("%s.templ", getComponentFileName(r.ComponentName))
}

// getGoFileName возвращает имя Go файла
func (r *ConversionResult) getGoFileName() string {
	return fmt.Sprintf("%s_controller.go", getComponentFileName(r.ComponentName))
}

// getJSFileName возвращает имя JavaScript файла
func (r *ConversionResult) getJSFileName() string {
	return fmt.Sprintf("%s.js", getComponentFileName(r.ComponentName))
}

// getComponentFileName возвращает имя файла компонента в нижнем регистре
func getComponentFileName(componentName string) string {
	var result bytes.Buffer

	for i, c := range componentName {
		if i > 0 && c >= 'A' && c <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(c)
	}

	return strings.ToLower(result.String())
}

// GetSummary возвращает краткую информацию о результате конвертации
func (r *ConversionResult) GetSummary() map[string]interface{} {
	files := make([]string, 0, 3)

	if r.TemplFile != "" {
		files = append(files, r.getTemplFileName())
	}
	if r.GoController != "" {
		files = append(files, r.getGoFileName())
	}
	if r.HtmxJS != "" {
		files = append(files, r.getJSFileName())
	}

	return map[string]interface{}{
		"componentName": r.ComponentName,
		"sourceFile":    r.SourceFile,
		"convertedAt":   r.ConvertedAt,
		"files":         files,
		"settings":      r.Settings,
	}
}
