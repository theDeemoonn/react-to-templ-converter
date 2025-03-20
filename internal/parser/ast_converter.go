package parser

import (
	"encoding/json"
	"fmt"
	"react-to-templ-converter/internal/models"
)

// ASTConverter преобразует AST из Node.js парсера в структуры Go
type ASTConverter struct{}

// NewASTConverter создает новый экземпляр конвертера AST
func NewASTConverter() *ASTConverter {
	return &ASTConverter{}
}

// ConvertJSONToComponent преобразует JSON-представление React компонента в Go структуру
func (c *ASTConverter) ConvertJSONToComponent(jsonData []byte) (*models.ReactComponent, error) {
	var component models.ReactComponent

	if err := json.Unmarshal(jsonData, &component); err != nil {
		return nil, fmt.Errorf("ошибка десериализации JSON: %w", err)
	}

	// Проверяем, что компонент имеет имя
	if component.Name == "" {
		return nil, fmt.Errorf("компонент не имеет имени")
	}

	// Проверяем наличие JSX
	if component.JSX == nil {
		return nil, fmt.Errorf("компонент не содержит JSX элементов")
	}

	return &component, nil
}

// ValidateComponent проверяет структуру компонента на корректность
func (c *ASTConverter) ValidateComponent(component *models.ReactComponent) error {
	// Проверка имени компонента
	if component.Name == "" {
		return fmt.Errorf("имя компонента не может быть пустым")
	}

	// Проверка JSX
	if component.JSX == nil {
		return fmt.Errorf("JSX элемент не может быть пустым")
	}

	// Проверка пропсов
	for i, prop := range component.Props {
		if prop.Name == "" {
			return fmt.Errorf("prop #%d имеет пустое имя", i+1)
		}
	}

	// Проверка состояний
	for i, state := range component.State {
		if state.Name == "" {
			return fmt.Errorf("state #%d имеет пустое имя", i+1)
		}
		if state.Setter == "" {
			return fmt.Errorf("state #%d не имеет setter-функции", i+1)
		}
	}

	return nil
}

// EnrichComponent дополняет компонент недостающей информацией
func (c *ASTConverter) EnrichComponent(component *models.ReactComponent) {
	// Устанавливаем типы для состояний на основе начальных значений
	for i, state := range component.State {
		if state.Type == "" && state.InitialValue != nil {
			component.State[i].Type = inferTypeFromValue(state.InitialValue)
		}
	}

	// Дополнение пропсов значениями по умолчанию для отсутствующих
	for i, prop := range component.Props {
		if !prop.Required && prop.DefaultValue == nil {
			component.Props[i].DefaultValue = getDefaultValueForType(prop.Type)
		}
	}
}

// Вспомогательные функции

// inferTypeFromValue определяет тип на основе значения
func inferTypeFromValue(value interface{}) string {
	switch value.(type) {
	case string:
		return "string"
	case float64, float32, int, int32, int64:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "any"
	}
}

// getDefaultValueForType возвращает значение по умолчанию для типа
func getDefaultValueForType(typeName string) interface{} {
	switch typeName {
	case "string":
		return ""
	case "number":
		return 0
	case "boolean":
		return false
	case "array":
		return []interface{}{}
	case "object":
		return map[string]interface{}{}
	default:
		return nil
	}
}
