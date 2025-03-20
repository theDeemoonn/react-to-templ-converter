package parser

import (
	"react-to-templ-converter/internal/models"
)

// ReactParser определяет интерфейс для парсинга React компонентов
type ReactParser interface {
	// ParseComponent принимает строку с React/TypeScript кодом и возвращает структуру компонента
	ParseComponent(code string) (*models.ReactComponent, error)

	// StartParser запускает парсер (если требуется)
	StartParser() error

	// StopParser останавливает парсер (если требуется)
	StopParser()
}

// ParserOption определяет опцию конфигурации для парсера
type ParserOption func(parser ReactParser)

// WithDebugMode включает режим отладки для парсера
func WithDebugMode(debug bool) ParserOption {
	return func(parser ReactParser) {
		if debugParser, ok := parser.(interface{ SetDebug(bool) }); ok {
			debugParser.SetDebug(debug)
		}
	}
}

// WithTimeout устанавливает таймаут для операций парсинга (в секундах)
func WithTimeout(timeoutSec int) ParserOption {
	return func(parser ReactParser) {
		if timeoutParser, ok := parser.(interface{ SetTimeout(int) }); ok {
			timeoutParser.SetTimeout(timeoutSec)
		}
	}
}
